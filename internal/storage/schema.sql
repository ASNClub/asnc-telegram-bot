PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS reputation (
    chat_id        INTEGER NOT NULL,
    user_id        INTEGER NOT NULL,
    username       TEXT    NOT NULL DEFAULT '',
    display_name   TEXT    NOT NULL DEFAULT '',
    score          INTEGER NOT NULL DEFAULT 0,
    positive_given INTEGER NOT NULL DEFAULT 0,
    negative_given INTEGER NOT NULL DEFAULT 0,
    updated_at     INTEGER NOT NULL,
    PRIMARY KEY (chat_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_reputation_chat_score
    ON reputation (chat_id, score DESC);

CREATE TABLE IF NOT EXISTS cooldown (
    chat_id        INTEGER NOT NULL,
    from_user_id   INTEGER NOT NULL,
    to_user_id     INTEGER NOT NULL,
    last_change_at INTEGER NOT NULL,
    PRIMARY KEY (chat_id, from_user_id, to_user_id)
);

CREATE TABLE IF NOT EXISTS known_users (
    chat_id    INTEGER NOT NULL,
    user_id    INTEGER NOT NULL,
    username   TEXT    NOT NULL DEFAULT '',
    first_name TEXT    NOT NULL DEFAULT '',
    last_name  TEXT    NOT NULL DEFAULT '',
    is_bot     INTEGER NOT NULL DEFAULT 0,
    seen_at    INTEGER NOT NULL,
    PRIMARY KEY (chat_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_known_users_chat_username
    ON known_users (chat_id, username);
