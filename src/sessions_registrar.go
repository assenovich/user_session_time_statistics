package main

import (
	"./settings"
	"time"
)

type Session struct {
	userId       string
	endTimestamp int64
	duration     int64
}

// Запрос на получение сессий, при помощи него же передаётся ответ
type SessionsInfo struct {
	userId    string // будем считать, что пустое имя пользователя соответствует запросу всех сессий
	durations []int64
}

type SessionRecord struct {
	next           *SessionRecord //TODO заменить список с отдельными аллокациями каждого элемента на что-то вроде std::dequeue из C++
	prev           *SessionRecord
	nextUserRecord *SessionRecord
	prevUserRecord *SessionRecord
	userInfo       *UserInfo
	endTimestamp   int64
	duration       int64
}

func (r *SessionRecord) initSessionRecordAsLatest() {
	r.next = r
	r.prev = r
	r.nextUserRecord = r
	r.prevUserRecord = r
	r.userInfo = nil
	r.endTimestamp = 0
	r.duration = 0
}

type UserInfo struct {
	latest SessionRecord
	userId string
}

type SessionsRecords struct {
	users  map[string]*UserInfo
	latest SessionRecord
}

// Здесь хранится список последних сессий
// Предположительно, статистические запросы будут гораздо более редкими,
// чем запросы о регистрации пользовательских сессий, т.е. в первую очередь
// быстрой должна быть регистрация пользовательской сессии.
// Сейчас, ради простоты реализации минимально работоспособного сервиса,
// средние и медианные значения считаются явным образом по списку сессий.
// Предположительно, у одного пользователя за период сбора статистики
// будет не очень много записей, поэтому их явная обработка при нахождении
// среднего/медианы будет не очень затратной.
// Но для медианного значения по всем сессиям это будет уже не так.
// Возможные направления дальнейшей оптимизации:
// - средние значения вообще не проблема держать заранее посчитанными и только
//   обновлять с приходом новых сессий; основная проблема с медианными значениями
// - возможно кеширование посчитанных средних значений с периодом инвалидации кеша ~10 секунд,
//   если запросы на статистику будут всё же частыми
// - для расчёта медианы по всем сессиям можно параллельно поддерживать
//   актуальной дополнительную структуру вроде min- и max-heap для времён сессий
//   (которая дополнительно занимает память порядка памяти списка всех сессий)
// - ещё вариант доп. структуры: т.к. среднее время сессии ~ 1 минута, то можно хранить гистограмму от 0 до 10 минут
//   с шагом в 10 секунд для сбора статистики коротких сессий, а хвост длинных сессий,
//   который будет предположительно небольшим, хранить явно
// - эти доп. структуры можно и также создавать по требованию для конкретного
//   пользователя, если у него также будет зарегистрировано слишком много сессий
// - для расчёта медианных значений можно смотреть в сторону приблизительных оценочных алгоритмов:
//   - https://www.cse.wustl.edu/~jain/papers/ftp/psqr.pdf
//   - https://arxiv.org/pdf/1407.1121v1.pdf
//   - http://web.ipac.caltech.edu/staff/fmasci/home/astro_refs/Remedian.pdf

// sessionsRegistrar регистрирует приходящие сессии и выдаёт по запросу список
// сессий конкретного пользователя или список всех сессий.
// Список сессий очищается от сессий, которые древнее, чем StatPeriod.
func sessionsRegistrar(sessions <-chan Session, sessionsInfoReqRes chan SessionsInfo) {
	sessionsRecors := SessionsRecords{users: make(map[string]*UserInfo)}
	sessionsRecors.latest.initSessionRecordAsLatest()
	for {
		// приоритетное чтение канала регистрации сессий: читается, пока есть что читать
		select {
		case session := <-sessions:
			sessionsRecors.processSession(session)
			continue
		default:
		}
		// низкоприоритетное чтение канал запросов списка сессий (потенциально долгая операция)
		select {
		case session := <-sessions:
			sessionsRecors.processSession(session)
			continue
		case request := <-sessionsInfoReqRes:
			if len(request.userId) == 0 {
				request.durations = sessionsRecors.createAllDurationsList()
			} else {
				request.durations = sessionsRecors.createDurationsList(request.userId)
			}
			sessionsInfoReqRes <- request
		}
	}
}

func (r *SessionsRecords) removeExpiredRecords() {
	currTimestamp := time.Now().UnixNano() / 1000000
	timestampThreshold := currTimestamp - settings.StatPeriod
	oldest := r.latest.next
	for oldest != &r.latest && oldest.endTimestamp < timestampThreshold {
		nextOldest := oldest.next
		nextOldest.prev = &r.latest
		r.latest.next = nextOldest
		userInfo := oldest.userInfo
		nextOldestUserRecord := oldest.nextUserRecord
		if nextOldestUserRecord == &userInfo.latest {
			delete(r.users, userInfo.userId)
		} else {
			nextOldestUserRecord.prevUserRecord = &userInfo.latest
			userInfo.latest.nextUserRecord = nextOldestUserRecord
		}
		oldest = nextOldest
	}
}

func (r *SessionsRecords) processSession(session Session) {
	if r.users[session.userId] == nil {
		userInfo := new(UserInfo)
		userInfo.latest.initSessionRecordAsLatest()
		userInfo.userId = session.userId
		r.users[session.userId] = userInfo
	}
	sessionRecord := new(SessionRecord)
	r.latest.prev.next = sessionRecord
	sessionRecord.prev = r.latest.prev
	sessionRecord.next = &r.latest
	r.latest.prev = sessionRecord
	userInfo := r.users[session.userId]
	sessionRecord.userInfo = userInfo
	userInfo.latest.prevUserRecord.nextUserRecord = sessionRecord
	sessionRecord.prevUserRecord = userInfo.latest.prevUserRecord
	sessionRecord.nextUserRecord = &userInfo.latest
	userInfo.latest.prevUserRecord = sessionRecord
	sessionRecord.endTimestamp = session.endTimestamp
	sessionRecord.duration = session.duration
	r.removeExpiredRecords()
}

func (r *SessionsRecords) createAllDurationsList() []int64 {
	result := make([]int64, 0)
	r.removeExpiredRecords()
	sessionRecord := r.latest.next
	for sessionRecord != &r.latest {
		result = append(result, sessionRecord.duration)
		sessionRecord = sessionRecord.next
	}
	return result
}

func (r *SessionsRecords) createDurationsList(userId string) []int64 {
	result := make([]int64, 0)
	r.removeExpiredRecords()
	if r.users[userId] == nil {
		delete(r.users, userId)
		return result
	}
	userInfo := r.users[userId]
	sessionRecord := userInfo.latest.nextUserRecord
	for sessionRecord != &userInfo.latest {
		result = append(result, sessionRecord.duration)
		sessionRecord = sessionRecord.nextUserRecord
	}
	return result
}
