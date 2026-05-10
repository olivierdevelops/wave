# polls

Create polls, vote on them, watch counts tick live via SSE.

## Run it

```sh
wave serve examples/apps/polls/server.yaml --port 8503
```

Open http://127.0.0.1:8503/app — create a poll, open it in two
tabs, vote in one, watch the other update.

## Try it

```sh
# Create poll
ID=$(curl -s -X POST localhost:8503/polls \
  -H 'content-type: application/json' \
  -d '{"title":"Best lang"}' | jq -r .id)

# Add options
for opt in go rust python; do
  curl -s -X POST localhost:8503/polls/$ID/options \
    -H 'content-type: application/json' \
    -d "{\"label\":\"$opt\"}"
done

# Read tallies
curl -s localhost:8503/polls/$ID | jq

# Vote (option_id from the poll's options_json)
curl -s -X POST localhost:8503/polls/$ID/vote \
  -H 'content-type: application/json' \
  -d '{"option_id":1}'

# Subscribe to live updates
curl -N localhost:8503/events/polls
```

## What to look at

- `connections.polls` — SSE broker auto-mounted at `/events/polls`.
- `type: stream-publish` route fans events to every subscriber.
- Each route's `execute:` is a single SQL statement so the
  driver's `LastInsertID` round-trips reliably.
- `json_group_array` packs the per-poll options into one column
  so the GET returns a single JSON document.

## Caveats

- No auth; anyone can vote any number of times.
- `/notify` is a separate POST so the broker fan-out is decoupled
  from the SQL UPDATE; the frontend hits both on each vote.

## What it shows off

SSE connections · stream-publish broker · array `inputs:` ·
`iterlist` template loop · JSON aggregation in SQL.
