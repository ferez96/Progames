package web

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"progames/internal/auth"
	"progames/internal/service"
)

func (fe *Frontend) practice(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.CurrentSession(r)
	user, _ := auth.CurrentUser(r)
	resp, err := fe.practiceSvc.GetPracticeData(service.GetPracticeDataRequest{UserID: user.ID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fe.render(w, r, "Practice", "practice", toPracticePage(resp, user.ID, session.CSRFToken, defaultCode, ""))
}

func (fe *Frontend) runPractice(w http.ResponseWriter, r *http.Request) {
	if !auth.ValidateCSRF(r) {
		http.Error(w, "invalid csrf token", http.StatusForbidden)
		return
	}
	user, _ := auth.CurrentUser(r)
	code, err := readCode(r)
	if err != nil {
		fe.practiceError(w, r, user.ID, err.Error())
		return
	}
	opponentID, err := strconv.ParseInt(r.FormValue("opponent_agent_id"), 10, 64)
	if err != nil {
		fe.practiceError(w, r, user.ID, "select a system opponent")
		return
	}
	resp, err := fe.practiceSvc.RunMatch(r.Context(), service.RunMatchRequest{
		UserID:     user.ID,
		Code:       code,
		OpponentID: opponentID,
	})
	if err != nil {
		fe.practiceError(w, r, user.ID, err.Error())
		return
	}
	target := fmt.Sprintf("/matches/%d", resp.MatchID)
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", target)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (fe *Frontend) practiceError(w http.ResponseWriter, r *http.Request, userID int64, msg string) {
	session, _ := auth.CurrentSession(r)
	resp, _ := fe.practiceSvc.GetPracticeData(service.GetPracticeDataRequest{UserID: userID})
	fe.renderStatus(w, r, http.StatusUnprocessableEntity, "Practice", "practice", toPracticePage(resp, userID, session.CSRFToken, defaultCode, msg))
}

func readCode(r *http.Request) (string, error) {
	if err := r.ParseMultipartForm(1 << 20); err == nil {
		file, _, err := r.FormFile("source_file")
		if err == nil {
			raw, readErr := io.ReadAll(file)
			closeErr := file.Close()
			if readErr != nil {
				return "", readErr
			}
			if closeErr != nil {
				return "", closeErr
			}
			return string(raw), nil
		}
	}
	if err := r.ParseForm(); err != nil {
		return "", err
	}
	return r.FormValue("source"), nil
}

const defaultCode = `package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		state := in.Text()
		for i, cell := range state {
			if cell == '.' {
				fmt.Printf("%d,%d\n", i%8+1, i/8+1)
				os.Stdout.Sync()
				break
			}
		}
		_ = strings.TrimSpace(state)
	}
}
`
