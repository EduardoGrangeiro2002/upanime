ALTER TABLE upscale_jobs ADD COLUMN type TEXT NOT NULL DEFAULT 'upscale'
    CHECK(type IN ('upscale', 'enhance'));
