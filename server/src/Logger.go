package src

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"strings"
	"time"
)

type Logger struct {
	db         *sql.DB
	logChannel chan logStruct
	loggerList []logStruct
	lastSent   time.Time
}

type logStruct struct {
	sessionId  string
	shareId    string
	action     string
	serverId   int
	time       time.Time
}

func MakeLogger() Logger {
	log := Logger{
		logChannel: make(chan logStruct),
		lastSent:   time.Now(),
	}

	go log.StartLogger()
	return log
}

func (l *Logger) LOG(sessionId, shareId, action string, time time.Time, serverId int) {
	l.logChannel <- logStruct{
		sessionId:  sessionId,
		shareId:    shareId,
		action:     action,
		time:       time,
		serverId:   serverId,
	}
}

const rowSQL = "(?, ?, ?, ?, ?, ?)"

func (l *Logger) StartLogger() {
	db, err := sql.Open("mysql", "bachelor:bachelor@tcp(therealflamingo.tk:3306)/bachelor_logging")
	if err != nil {
		panic("LOGGER COULD NOT CONNECT TO DATABASE")
	}
	db.SetMaxIdleConns(0)
	l.loggerList = []logStruct{}
	for {
		select {
		case tmp := <-l.logChannel:
			l.loggerList = append(l.loggerList, tmp)
		case <-time.After(5 * time.Second):
		}

		if time.Now().Unix()-l.lastSent.Unix() > 10 || len(l.loggerList) > 5000 {
			sqlStr := "INSERT INTO Log " +
				"(_TIMESTAMP, _SESSION_ID, _SERVER_ID, _ACTION, _SHARE_ID, _PROTOCOL_TYPE) VALUES "
			var val []interface{}
			var inserts []string

			for _, v := range l.loggerList {
				inserts = append(inserts, rowSQL)
				val = append(val, v.time.UnixNano(), v.sessionId, v.serverId, v.action, v.shareId, ProtocolType)
			}

			if len(inserts) != 0 {
				sqlStr += strings.Join(inserts, ",")
				stmtIns, err := db.Prepare(sqlStr)
				if err != nil {
					time.Sleep(100)
					continue
				}
				stmtIns.Exec(val...)
			}

			l.loggerList = []logStruct{}
			l.lastSent = time.Now()
		}
	}
}
