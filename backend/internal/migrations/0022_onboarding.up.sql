-- Financial setup is private to a profile. Drafts never count as booked money.
CREATE TABLE profile_onboarding (
    profile_id UUID PRIMARY KEY REFERENCES profiles(id) ON DELETE CASCADE,
    version INTEGER NOT NULL DEFAULT 1 CHECK (version = 1),
    status TEXT NOT NULL CHECK (status IN ('draft','skipped','completed')),
    step INTEGER NOT NULL DEFAULT 0 CHECK (step BETWEEN 0 AND 5),
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    draft JSONB NOT NULL CHECK (jsonb_typeof(draft) = 'object'),
    result JSONB CHECK (result IS NULL OR jsonb_typeof(result) = 'object'),
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((status = 'completed') = (completed_at IS NOT NULL)),
    CHECK ((status = 'completed') = (result IS NOT NULL))
);
