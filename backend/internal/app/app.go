package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"

	"sportsbook/backend/internal/betting"
	"sportsbook/backend/internal/line"
	"sportsbook/backend/internal/store"
)

// currencyRUB is the demo player's currency; the schema keeps it per player
// because a real deployment could serve more than one.
const currencyRUB = "RUB"

// Limits are the public stake limits.
type Limits struct {
	MinStakeMinor int64 `json:"minStakeMinor"`
	MaxStakeMinor int64 `json:"maxStakeMinor"`
}

// Deps wires the HTTP app.
type Deps struct {
	Service *betting.Service
	Engine  *line.Engine
	Limits  Limits
	Now     func() time.Time
	// Pool is optional: when set, /api/healthz pings the database.
	Pool *pgxpool.Pool
}

// New builds the Fiber app with every /api route.
func New(d Deps) *fiber.App {
	if d.Now == nil {
		d.Now = time.Now
	}
	s := &server{deps: d}

	app := fiber.New(fiber.Config{
		AppName:      "overtime",
		ErrorHandler: errorHandler,
	})
	app.Use(recover.New())

	api := app.Group("/api")
	api.Get("/session", s.session)
	api.Get("/line", s.line)
	api.Get("/bets", s.bets)
	api.Post("/bets", s.placeBet)
	api.Post("/bets/:id/settle-demo", s.settleDemoBet)
	api.Get("/stream", s.stream)
	api.Get("/healthz", s.healthz)
	return app
}

type server struct {
	deps Deps
}

func (s *server) session(c *fiber.Ctx) error {
	player, err := s.deps.Service.Session(c.UserContext())
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return apiError(c, fiber.StatusNotFound, "not_found", "Игрок не найден", nil)
		}
		return internalError(c, err)
	}
	return c.JSON(fiber.Map{
		"player":     player,
		"limits":     s.deps.Limits,
		"serverTime": s.deps.Now().UTC().Format(time.RFC3339),
	})
}

func (s *server) line(c *fiber.Ctx) error {
	return c.JSON(s.deps.Engine.Snapshot())
}

func (s *server) bets(c *fiber.Ctx) error {
	// Zero means "use the store's default"; the store also caps the maximum.
	limit := 0
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return apiError(c, fiber.StatusUnprocessableEntity, "validation_error", "limit должен быть числом", nil)
		}
		limit = n
	}

	bets, err := s.deps.Service.History(c.UserContext(), limit)
	if err != nil {
		return internalError(c, err)
	}
	return c.JSON(fiber.Map{"bets": bets})
}

type placeBetRequest struct {
	MatchID            string `json:"matchId"`
	Outcome            string `json:"outcome"`
	StakeMinor         int64  `json:"stakeMinor"`
	ExpectedPriceMilli int64  `json:"expectedPriceMilli"`
	IdempotencyKey     string `json:"idempotencyKey"`
}

func (s *server) placeBet(c *fiber.Ctx) error {
	var req placeBetRequest
	if err := c.BodyParser(&req); err != nil {
		return apiError(c, fiber.StatusUnprocessableEntity, "validation_error", "Некорректный JSON", nil)
	}

	key := strings.TrimSpace(req.IdempotencyKey)
	if req.MatchID == "" || req.StakeMinor <= 0 || req.ExpectedPriceMilli <= 0 ||
		len(key) < 8 || len(key) > 100 {
		return apiError(c, fiber.StatusUnprocessableEntity, "validation_error", "Проверьте параметры ставки", nil)
	}
	switch req.Outcome {
	case "home", "draw", "away":
	default:
		return apiError(c, fiber.StatusUnprocessableEntity, "validation_error", "Некорректный исход", nil)
	}

	result, err := s.deps.Service.Place(c.UserContext(), betting.PlaceInput{
		MatchID:            req.MatchID,
		Outcome:            req.Outcome,
		StakeMinor:         req.StakeMinor,
		ExpectedPriceMilli: req.ExpectedPriceMilli,
		IdempotencyKey:     key,
	})
	if err != nil {
		return bettingError(c, err)
	}

	status := fiber.StatusCreated
	if result.Replayed {
		status = fiber.StatusOK
	}
	return c.Status(status).JSON(fiber.Map{
		"bet":          result.Bet,
		"balanceMinor": result.BalanceMinor,
		"currency":     currencyRUB,
		"replayed":     result.Replayed,
	})
}

func (s *server) settleDemoBet(c *fiber.Ctx) error {
	result, err := s.deps.Service.SettleDemo(c.UserContext(), c.Params("id"))
	if err != nil {
		return bettingError(c, err)
	}
	return c.JSON(fiber.Map{
		"bet":          result.Bet,
		"balanceMinor": result.BalanceMinor,
		"currency":     currencyRUB,
		"won":          result.Won,
	})
}

func (s *server) stream(c *fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set(fiber.HeaderConnection, "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	// Capture the server's shutdown channel while the request context is still
	// valid: fasthttp runs the stream writer in its own goroutine and forbids
	// touching the RequestCtx from there.
	done := c.Context().Done()

	// Fiber v2 exposes the fasthttp stream writer through the request context;
	// the bufio.Writer is flushed after every event below.
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Subscribe before taking the snapshot: an update that lands in
		// between is either skipped by seq or delivered as the next line event.
		snapshots, cancelSnapshots := s.deps.Engine.Subscribe()
		defer cancelSnapshots()
		balances, cancelBalances := s.deps.Service.SubscribeBalance()
		defer cancelBalances()

		current := s.deps.Engine.Snapshot()
		lastSeq := current.Seq
		if !writeEvent(w, "snapshot", current.Seq, current) {
			return
		}

		heartbeat := time.NewTicker(20 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case <-done:
				return
			case next := <-snapshots:
				if next.Seq <= lastSeq {
					continue
				}
				lastSeq = next.Seq
				if !writeEvent(w, "line", next.Seq, next) {
					return
				}
			case event := <-balances:
				if !writeEvent(w, "balance", -1, event) {
					return
				}
			case <-heartbeat.C:
				if _, err := w.WriteString(": ping\n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
	return nil
}

func (s *server) healthz(c *fiber.Ctx) error {
	if s.deps.Pool != nil {
		ctx, cancel := context.WithTimeout(c.UserContext(), time.Second)
		defer cancel()
		if err := s.deps.Pool.Ping(ctx); err != nil {
			slog.Error("healthz: database ping failed", "err", err)
			return apiError(c, fiber.StatusServiceUnavailable, "internal", "База данных недоступна", nil)
		}
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

// writeEvent serializes one SSE event with a single-line JSON payload. A
// negative id omits the id line. It returns false when the client is gone.
func writeEvent(w *bufio.Writer, name string, id int64, payload any) bool {
	data, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	if _, err := fmt.Fprintf(w, "event: %s\n", name); err != nil {
		return false
	}
	if id >= 0 {
		if _, err := fmt.Fprintf(w, "id: %d\n", id); err != nil {
			return false
		}
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return false
	}
	return w.Flush() == nil
}

func bettingError(c *fiber.Ctx, err error) error {
	var oddsChanged *betting.OddsChangedError
	switch {
	case errors.As(err, &oddsChanged):
		return apiError(c, fiber.StatusConflict, "odds_changed", "Коэффициент изменился", fiber.Map{
			"expectedPriceMilli": oddsChanged.Expected,
			"priceMilli":         oddsChanged.Current,
		})
	case errors.Is(err, betting.ErrMarketNotFound):
		return apiError(c, fiber.StatusNotFound, "market_not_found", "Матч или исход не найден", nil)
	case errors.Is(err, betting.ErrBetNotFound):
		return apiError(c, fiber.StatusNotFound, "bet_not_found", "Ставка не найдена", nil)
	case errors.Is(err, betting.ErrBetNotPending):
		return apiError(c, fiber.StatusConflict, "bet_not_pending", "Ставка уже рассчитана", nil)
	case errors.Is(err, betting.ErrInsufficientFunds):
		return apiError(c, fiber.StatusUnprocessableEntity, "insufficient_funds", "Недостаточно средств", nil)
	case errors.Is(err, betting.ErrStakeOutOfRange):
		return apiError(c, fiber.StatusUnprocessableEntity, "stake_out_of_range", "Ставка вне допустимого диапазона", nil)
	case errors.Is(err, betting.ErrInvalidRequest):
		return apiError(c, fiber.StatusUnprocessableEntity, "validation_error", "Проверьте параметры ставки", nil)
	default:
		return internalError(c, err)
	}
}

func internalError(c *fiber.Ctx, err error) error {
	slog.Error("request failed", "method", c.Method(), "path", c.Path(), "err", err)
	return apiError(c, fiber.StatusInternalServerError, "internal", "Внутренняя ошибка", nil)
}

func apiError(c *fiber.Ctx, status int, code, message string, details fiber.Map) error {
	errObject := fiber.Map{"code": code, "message": message}
	if len(details) > 0 {
		errObject["details"] = details
	}
	return c.Status(status).JSON(fiber.Map{"error": errObject})
}

// errorHandler renders framework errors (unknown route, panic recovery) in
// the same envelope as application errors instead of Fiber's plain text.
func errorHandler(c *fiber.Ctx, err error) error {
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		switch {
		case fiberErr.Code == fiber.StatusNotFound:
			return apiError(c, fiber.StatusNotFound, "not_found", "Не найдено", nil)
		case fiberErr.Code < fiber.StatusInternalServerError:
			return apiError(c, fiberErr.Code, "validation_error", "Некорректный запрос", nil)
		}
	}
	return internalError(c, err)
}
