package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"progames/internal/auth"
	"progames/internal/service"
)

func (fe *Frontend) matchSummary(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.CurrentUser(r)
	matchID, ok := parseMatchID(w, r)
	if !ok {
		return
	}
	matchResp, err := fe.matchSvc.GetMatch(service.GetMatchRequest{UserID: user.ID, MatchID: matchID})
	if err != nil {
		respondMatchErr(w, err)
		return
	}
	gamesResp, err := fe.gameSvc.ListGames(service.ListGamesRequest{UserID: user.ID, MatchID: matchID})
	if err != nil {
		respondMatchErr(w, err)
		return
	}
	fe.render(w, r, fmt.Sprintf("Match #%d", matchID), "match", toSummaryPage(matchResp, gamesResp))
}

func (fe *Frontend) matchLogs(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.CurrentUser(r)
	matchID, ok := parseMatchID(w, r)
	if !ok {
		return
	}
	resp, err := fe.matchSvc.GetExecutionLog(service.GetExecutionLogRequest{UserID: user.ID, MatchID: matchID})
	if err != nil {
		respondMatchErr(w, err)
		return
	}
	fe.render(w, r, fmt.Sprintf("Match #%d Logs", matchID), "logs", logsPageData{
		MatchID: matchID,
		Content: resp.Content,
		HasLog:  resp.Content != "",
	})
}

func (fe *Frontend) matchGames(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.CurrentUser(r)
	matchID, ok := parseMatchID(w, r)
	if !ok {
		return
	}
	gamesResp, err := fe.gameSvc.ListGames(service.ListGamesRequest{UserID: user.ID, MatchID: matchID})
	if err != nil {
		respondMatchErr(w, err)
		return
	}
	names := agentNameMap(gamesResp.AgentA, gamesResp.AgentB)
	rows := make([]gameRow, len(gamesResp.Games))
	for i, g := range gamesResp.Games {
		rows[i] = toGameRow(g, names)
	}
	page := gamesPageData{MatchID: matchID, Games: rows}
	if len(gamesResp.Games) > 0 {
		req := service.GetGameRequest{UserID: user.ID, MatchID: matchID, GameID: gamesResp.Games[0].ID}
		if gameResp, err := fe.gameSvc.GetGame(req); err == nil {
			v := toGameView(gameResp)
			page.InitialGame = &v
		}
	}
	fe.render(w, r, fmt.Sprintf("Match #%d — Games", matchID), "games", page)
}

func (fe *Frontend) matchGameReplay(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.CurrentUser(r)
	matchID, ok := parseMatchID(w, r)
	if !ok {
		return
	}
	gameID, err := strconv.ParseInt(chi.URLParam(r, "gid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid game id", http.StatusBadRequest)
		return
	}
	resp, err := fe.gameSvc.GetGame(service.GetGameRequest{UserID: user.ID, MatchID: matchID, GameID: gameID})
	if err != nil {
		respondMatchErr(w, err)
		return
	}
	fe.render(w, r, fmt.Sprintf("Game #%d Replay", gameID), "game_replay", toGameView(resp))
}

func parseMatchID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	matchID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid match id", http.StatusBadRequest)
		return 0, false
	}
	return matchID, true
}

func respondMatchErr(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
