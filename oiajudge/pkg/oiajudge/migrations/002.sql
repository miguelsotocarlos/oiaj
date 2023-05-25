ALTER TABLE oia_task_score DROP CONSTRAINT oia_task_score_pkey;;

UPDATE oia_task_score
SET user_id = (SELECT cms_user_id FROM oia_user WHERE oia_user.id = oia_task_score.user_id);;

ALTER TABLE oia_task_score ADD PRIMARY KEY (user_id, task_id);;


TRUNCATE TABLE oia_tokens;;


ALTER TABLE oia_user DROP CONSTRAINT oia_user_pkey CASCADE;

UPDATE oia_user
SET id = oia_user.cms_user_id;;

ALTER TABLE oia_user DROP COLUMN cms_user_id;;

ALTER TABLE oia_user ADD PRIMARY KEY (id);;

ALTER TABLE oia_submissions ADD CONSTRAINT fk_user_id
        FOREIGN KEY(user_id)
            REFERENCES oia_user(id);;


ALTER TABLE oia_tokens ADD CONSTRAINT fk_user_id
        FOREIGN KEY(user_id)
            REFERENCES oia_user(id);;

ALTER TABLE oia_task_score ADD CONSTRAINT fk_user_id
        FOREIGN KEY(user_id)
            REFERENCES oia_user(id)