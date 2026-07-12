ALTER TABLE animes ADD COLUMN genres TEXT NOT NULL DEFAULT '[]';

INSERT INTO scrapers (name, domain, script_path)
VALUES ('upload', 'upload', '');
