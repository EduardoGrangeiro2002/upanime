ALTER TABLE animes DROP COLUMN genres;

DELETE FROM scrapers WHERE domain = 'upload';
