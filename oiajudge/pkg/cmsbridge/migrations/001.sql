CREATE TABLE IF NOT EXISTS event_queue (
    id SERIAL PRIMARY KEY,
    foreign_id INTEGER NOT NULL,
    object_type TEXT NOT NULL,
    seen BOOLEAN DEFAULT false NOT NULL
)

;;

INSERT INTO event_queue(foreign_id, object_type)
SELECT
    id AS foreign_id,
    'task' AS object_type
FROM tasks

;;

-- Submission Result
CREATE OR REPLACE FUNCTION register_submission_result_change()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    IF NEW.submission_id IS NOT NULL THEN
        INSERT INTO event_queue(foreign_id, object_type) VALUES (NEW.submission_id, 'submission');
    END IF;
    RETURN NEW;
END;
$$

;;

CREATE OR REPLACE TRIGGER submission_result_change
    BEFORE UPDATE OR INSERT
    ON submission_results
    FOR EACH ROW
    EXECUTE PROCEDURE register_submission_result_change();

-- Submission
CREATE OR REPLACE FUNCTION register_submission_change()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    IF NEW.id IS NOT NULL THEN
        INSERT INTO event_queue(foreign_id, object_type) VALUES (NEW.id, 'submission');
    END IF;
    RETURN NEW;
END;
$$

;;

CREATE OR REPLACE TRIGGER submission_change
    BEFORE UPDATE OR INSERT
    ON submissions
    FOR EACH ROW
    EXECUTE PROCEDURE register_submission_change();

-- Task
CREATE OR REPLACE FUNCTION register_task_change()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    IF NEW.id IS NOT NULL THEN
        INSERT INTO event_queue(foreign_id, object_type) VALUES (NEW.id, 'task');
    END IF;
    RETURN NEW;
END;
$$

;;

CREATE OR REPLACE TRIGGER task_change
    BEFORE UPDATE OR INSERT
    ON tasks
    FOR EACH ROW
    EXECUTE PROCEDURE register_task_change();

;;

-- Task
CREATE OR REPLACE FUNCTION notify_notification()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    NOTIFY event_queue;
    RETURN NEW;
END;
$$

;;

CREATE OR REPLACE TRIGGER notify_notification_trigger
    BEFORE INSERT
    ON event_queue
    FOR EACH ROW
    EXECUTE PROCEDURE notify_notification();