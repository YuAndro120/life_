-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE sources (
    id           bigserial PRIMARY KEY,
    kind         text        NOT NULL CHECK (kind IN ('rss', 'tg', 'gov', 'site')),
    handle       text        NOT NULL,
    url          text        NOT NULL,
    title        text        NOT NULL,
    topic_hint   text,
    -- Источники из реестров нежелательных организаций и иноагентов в каталог не входят.
    legal_status text        NOT NULL DEFAULT 'ok' CHECK (legal_status IN ('ok', 'excluded')),
    active       boolean     NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (kind, handle)
);

CREATE TABLE stories (
    id            bigserial PRIMARY KEY,
    first_seen_at timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    topic         text NOT NULL CHECK (topic IN (
        'economy', 'finance', 'law', 'tech_ai', 'city', 'health', 'education', 'transport',
        'housing', 'science', 'culture', 'sport', 'showbiz', 'crypto',
        'politics', 'crime', 'incidents', 'disasters')),
    info_type     text NOT NULL CHECK (info_type IN ('fact', 'official', 'opinion', 'forecast', 'rumor')),
    heaviness     text NOT NULL CHECK (heaviness IN ('neutral', 'tense', 'heavy')),
    title_neutral text NOT NULL,
    summary       text NOT NULL,
    meaning       text,
    post_count    integer NOT NULL DEFAULT 0,
    source_count  integer NOT NULL DEFAULT 0,
    centroid      vector(1024),
    region_code   text,
    status        text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published'))
);

CREATE TABLE posts (
    id           bigserial PRIMARY KEY,
    source_id    bigint      NOT NULL REFERENCES sources (id),
    external_id  text        NOT NULL,
    url          text        NOT NULL,
    published_at timestamptz NOT NULL,
    text         text        NOT NULL DEFAULT '',
    lang         text        NOT NULL DEFAULT 'ru',
    is_ad        boolean     NOT NULL DEFAULT false,
    embedding    vector(1024),
    story_id     bigint REFERENCES stories (id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_id, external_id)
);

CREATE TABLE story_posts (
    story_id bigint NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    post_id  bigint NOT NULL REFERENCES posts (id) ON DELETE CASCADE,
    PRIMARY KEY (story_id, post_id)
);

CREATE TABLE law_changes (
    id            bigserial PRIMARY KEY,
    title         text   NOT NULL,
    what_changed  text   NOT NULL,
    who_affected  text   NOT NULL,
    actions       jsonb  NOT NULL DEFAULT '[]'::jsonb,
    audience_tags text[] NOT NULL DEFAULT '{}',
    region_code   text,
    status        text   NOT NULL CHECK (status IN ('introduced', 'passed', 'signed', 'in_force')),
    introduced_at date,
    passed_at     date,
    signed_at     date,
    effective_at  date,
    official_url  text,
    bill_url      text,
    act_number    text,
    -- В API попадают только проверенные вручную (lawtool review).
    verified      boolean NOT NULL DEFAULT false,
    verified_at   timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX posts_published_at_idx ON posts (published_at);
CREATE INDEX posts_story_id_idx ON posts (story_id);
CREATE INDEX stories_updated_at_idx ON stories (updated_at);
CREATE INDEX law_changes_effective_at_idx ON law_changes (effective_at);
CREATE INDEX posts_embedding_hnsw_idx ON posts USING hnsw (embedding vector_cosine_ops);
CREATE INDEX stories_centroid_hnsw_idx ON stories USING hnsw (centroid vector_cosine_ops);

-- +goose Down
DROP TABLE law_changes;
DROP TABLE story_posts;
DROP TABLE posts;
DROP TABLE stories;
DROP TABLE sources;
