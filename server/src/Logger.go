package src

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	db         *sql.DB
	logChannel chan logStruct
	loggerList []logStruct
	lastSent   time.Time
	lock       sync.Mutex
}

type logStruct struct {
	sessionId string
	shareId   string
	action    string
	serverId  int
	time      time.Time
}

func MakeLogger() Logger {
	log := Logger{
		logChannel: make(chan logStruct),
		lastSent:   time.Now(),
	}

	go log.StartLogger()
	return log
}

func (l *Logger) LOG(sessionId, shareId, action string, serverId int, time time.Time) {
	l.logChannel <- logStruct{
		sessionId: sessionId,
		shareId:   shareId,
		action:    action,
		time:      time,
		serverId:  serverId,
	}
}

const rowSQL = "(?, ?, ?, ?, ?, ?)"

func (l *Logger) StartLogger() {
	db, err := sql.Open("mysql", "bachelor:bachelor@tcp(68.183.77.224:3306)/bachelor_logging")
	if err != nil {
		panic("LOGGER COULD NOT CONNECT TO DATABASE")
	}
	db.SetMaxIdleConns(0)
	l.loggerList = []logStruct{}
	for {
		select {
		case tmp := <-l.logChannel:
			l.loggerList = append(l.loggerList, tmp)
		case <-time.After(1 * time.Second):
		}

		if time.Now().Unix()-l.lastSent.Unix() > 10 || len(l.loggerList) > 5000 {
			resToSend := make([]logStruct, 5000)
			var rest []logStruct
			cpyAmount := copy(resToSend, l.loggerList)
			if cpyAmount > 5000 {
				rest = append(rest, l.loggerList[5000:]...)
			}
			if cpyAmount == 0 {
				continue
			}

			//fmt.Println(len(l.loggerList), time.Now().Unix()-l.lastSent.Unix())
			sqlStr := "INSERT INTO Log " +
				"(_TIMESTAMP, _SESSION_ID, _SERVER_ID, _ACTION, _SHARE_ID, _PROTOCOL_TYPE) VALUES "
			var val []interface{}
			var inserts []string

			for i := 0; i < cpyAmount; i++ {
				v := resToSend[i]
				inserts = append(inserts, rowSQL)
				val = append(val, v.time.UnixNano(), v.sessionId, v.serverId, v.action, v.shareId, ProtocolType)
			}

			if len(inserts) != 0  && val != nil{
				sqlStr += strings.Join(inserts, ",")
				stmtIns, err := db.Prepare(sqlStr)
				if err != nil {
					fmt.Println(err)
					time.Sleep(1000)
					continue
				}
				//fmt.Println(val)
				stmtIns.Exec(val...)
				stmtIns.Close()
				//fmt.Println(err, res)
			}

			l.loggerList = rest
			l.lastSent = time.Now()
		}
	}
}
