-- Put all submissions into the event queue
INSERT INTO event_queue(foreign_id, object_type) SELECT id, 'submission' FROM submissions;