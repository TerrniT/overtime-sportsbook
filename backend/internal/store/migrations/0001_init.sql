-- Initial schema. Applied at startup; every statement is idempotent.
create table if not exists players (
  id            uuid primary key,
  name          text not null,
  currency      text not null default 'RUB',
  balance_minor bigint not null check (balance_minor >= 0),
  created_at    timestamptz not null default now()
);

create table if not exists bets (
  id                     uuid primary key,
  player_id              uuid not null references players(id),
  idempotency_key        text not null,
  match_id               text not null,
  match_label            text not null,
  outcome                text not null check (outcome in ('home', 'draw', 'away')),
  outcome_label          text not null,
  stake_minor            bigint not null check (stake_minor > 0),
  price_milli            bigint not null check (price_milli between 1050 and 20000),
  potential_payout_minor bigint not null check (potential_payout_minor >= stake_minor),
  status                 text not null default 'pending'
                           check (status in ('pending', 'won', 'lost', 'void')),
  placed_at              timestamptz not null default now(),
  unique (player_id, idempotency_key)
);

create index if not exists bets_player_placed_idx
  on bets (player_id, placed_at desc);

create table if not exists ledger_entries (
  id                  bigserial primary key,
  player_id           uuid not null references players(id),
  bet_id              uuid references bets(id),
  kind                text not null
                        check (kind in ('deposit', 'bet_stake', 'bet_win', 'bet_refund')),
  amount_minor        bigint not null,
  balance_after_minor bigint not null check (balance_after_minor >= 0),
  created_at          timestamptz not null default now()
);

create index if not exists ledger_player_idx
  on ledger_entries (player_id, created_at desc);

-- One demo player. Never resets an existing balance.
insert into players (id, name, currency, balance_minor)
values ('11111111-1111-4111-8111-111111111111', 'Игрок', 'RUB', 2500000)
on conflict (id) do nothing;

insert into ledger_entries (player_id, kind, amount_minor, balance_after_minor)
select p.id, 'deposit', p.balance_minor, p.balance_minor
from players p
where p.id = '11111111-1111-4111-8111-111111111111'
  and not exists (
    select 1 from ledger_entries l where l.player_id = p.id and l.kind = 'deposit'
  );
