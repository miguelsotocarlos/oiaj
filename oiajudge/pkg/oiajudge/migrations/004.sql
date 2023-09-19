-- Add attachment field to oia_task table
ALTER TABLE oia_task ADD COLUMN attachments TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[];