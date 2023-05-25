ALTER TABLE oia_task_score
DISABLE TRIGGER ALL;;

UPDATE oia_task_score
SET user_id = (SELECT cms_user_id FROM oia_user WHERE oia_user.id = oia_task_score.user_id);;

ALTER TABLE oia_task_score
ENABLE TRIGGER ALL;;



ALTER TABLE oia_tokens
DISABLE TRIGGER ALL;;

UPDATE oia_tokens oia_tokens
SET user_id = (SELECT cms_user_id FROM oia_user WHERE oia_user.id = oia_tokens.user_id);;

ALTER TABLE oia_tokens
ENABLE TRIGGER ALL;;



ALTER TABLE oia_user
DISABLE TRIGGER ALL;;

UPDATE oia_user
SET id = oia_user.cms_user_id;;

ALTER TABLE oia_user DROP COLUMN cms_user_id;;

ALTER TABLE oia_user
ENABLE TRIGGER ALL
