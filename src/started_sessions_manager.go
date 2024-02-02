package main

type UserSessionEventType int

const (
	SessionStartedEventType UserSessionEventType = 1 + iota
	SessionEndedEventType
)

type UserSessionEventData struct {
	User_id    string `json:"user_id"`
	Session_id string `json:"session_id"`
	Timestamp  int64  `json:"timestamp,string"`
}

type UserSessionEvent struct {
	eventType UserSessionEventType
	data      UserSessionEventData
}

// Менеджер открытых и ещё не закрытых пользовательских сессий.
// ? в условии задачи не ясно, может ли быть у одного пользователся несколько открытых сессий,
//   здесь выбран вариант, когда это легальная ситуация
// ? также неясно, что делать при получении несогласованных событий о начале и конце сессии,
//   здесь молчаливо игнорируется закрытие ещё не открытой сессии и игнорируется предыдущее
//   открытие сессии при повторном её открытии
func startedSessionsManager(userSessionEvents <-chan UserSessionEvent, sessions chan<- Session) {
	// Timestamp'ы открытых сессий пользователей (время начала сессии)
	startedSessions := make(map[string]int64)

	//TODO следует добавить механизм периодического выкидывания давно открытых сессий,
	// которые по каким-то причинам уже не будут никогда закрыты

	//TODO сформулировать, что надо делать, когда открытых сессий становится слишком много

	for event := range userSessionEvents {
		// Здесь также предполагается ради простоты, что в имени пользователя или сессии не может
		// встречаться символ '/', поэтому уникальным глобальным идентификатором сессии может служить
		// User_id + '/' + Session_id. Если это не так, то можно, например, экранировать символ '/'
		globalSessionId := event.data.User_id + "/" + event.data.Session_id

		switch event.eventType {
		case SessionStartedEventType:
			startedSessions[globalSessionId] = event.data.Timestamp
		case SessionEndedEventType:
			startTimestamp, ok := startedSessions[globalSessionId]
			if ok {
				endTimestamp := event.data.Timestamp
				duration := endTimestamp - startTimestamp
				if duration > 0 { // также тихо игнорируем сессии с некорректной длительностью
					sessions <- Session{
						userId:       event.data.User_id,
						endTimestamp: endTimestamp,
						duration:     duration,
					}
				}
			}
			delete(startedSessions, globalSessionId)
		default:
			panic("Unexpected session event type")
		}
	}
}
