package cmsbridge

import (
	"log"

	"github.com/carlosmiguelsoto/oiajudge/pkg/bridge"
	"github.com/carlosmiguelsoto/oiajudge/pkg/store"
)

func GetAttachment(tx store.Transaction, tid bridge.Id, filename string) (attachment []byte, err error) {
	rows, err := tx.Query(`
			SELECT pg_largeobject.data
					FROM attachments
					INNER JOIN fsobjects ON attachments.digest = fsobjects.digest
					INNER JOIN pg_largeobject ON fsobjects.loid = pg_largeobject.loid
					WHERE attachments.task_id = $1 AND attachments.filename = $2
					ORDER BY pg_largeobject.pageno ASC;`,
		tid, filename)
	if err != nil {
		log.Printf("GetTask(): error getting attachment body: %s", err)
	}
	for rows.Next() {
		var page []byte
		err = rows.Scan(&page)
		if err != nil {
			return
		}
		attachment = append(attachment, page...)
	}
	return
}
