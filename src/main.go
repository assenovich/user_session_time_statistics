package main

import (
	"./settings"
	"encoding/json"
	"fmt"
	"net/http"
)

type statRequestProcessor func(string) int64

func main() {
	fmt.Println("Starting mean/median user session time statistic server...")

	userSessionEvents := make(chan UserSessionEvent, settings.UserSessionEventBufferSize)
	sessions := make(chan Session, settings.UserSessionEventBufferSize)
	sessionsInfoReqRes := make(chan SessionsInfo)

	// эта горутина конвертирует поток сообщений session{Started, Ended} в поток событий о сессиях, ...
	go startedSessionsManager(userSessionEvents, sessions)

	// ... который подхватывает эта горутина и поддерживает внутри себя актуальную информацию о последних сессиях, ...
	go sessionsRegistrar(sessions, sessionsInfoReqRes)

	// ... которую (по что в виде явного списка сессий) используют статистические функции
	// посредством getSessionsDurations
	getSessionsDurations := func(userId string) []int64 {
		sessionsInfoReqRes <- SessionsInfo{userId: userId}
		response := <-sessionsInfoReqRes
		return response.durations
	}

	addSessionEventHandler := func(urlPath string, eventType UserSessionEventType) {
		http.HandleFunc(urlPath, createSessionEventsHandler(urlPath, eventType, userSessionEvents))
	}
	addSessionEventHandler("/sessionStarted", SessionStartedEventType)
	addSessionEventHandler("/sessionEnded", SessionEndedEventType)

	addStatRequestHandler := func(urlPath string, processor statRequestProcessor) {
		http.HandleFunc(urlPath, createStatRequestHandler(urlPath, processor))
	}
	addStatRequestHandler("/meanTime", func(userId string) int64 {
		return calcMean(getSessionsDurations(userId))
	})
	addStatRequestHandler("/medianTime", func(userId string) int64 {
		return calcMedian(getSessionsDurations(userId))
	})

	if err := http.ListenAndServe(settings.ListenAddress, nil); err != nil {
		panic(err)
	}

	//TODO пока не предполагается какого-то штатного механизма остановки сервиса

	fmt.Println("END")
}

func createSessionEventsHandler(urlPath string, eventType UserSessionEventType, userSessionEvents chan<- UserSessionEvent) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != urlPath || r.Method != "POST" {
			return
		}
		decoder := json.NewDecoder(r.Body)
		var event UserSessionEvent
		event.eventType = eventType
		err := decoder.Decode(&event.data)
		if err != nil {
			return
		}
		userSessionEvents <- event
	}
}

func createStatRequestHandler(urlPath string, processor statRequestProcessor) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != urlPath || r.Method != "GET" {
			return
		}
		if err := r.ParseForm(); err != nil {
			return
		}
		fmt.Fprintf(w, "%d", processor(r.FormValue("user_id")))
	}
}
