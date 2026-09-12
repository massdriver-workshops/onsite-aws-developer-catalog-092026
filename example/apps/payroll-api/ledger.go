package main

import (
	"sort"
	"sync"
	"time"
)

// Event mirrors the contract published by timeoff-api.
type Event struct {
	Type       string `json:"type"`
	Version    int    `json:"version"`
	RequestID  int64  `json:"request_id"`
	EmployeeID int64  `json:"employee_id"`
	Employee   string `json:"employee"`
	Team       string `json:"team"`
	Kind       string `json:"kind"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	Days       int    `json:"days"`
	Status     string `json:"status"`
	DecidedBy  string `json:"decided_by,omitempty"`
	Namespace  string `json:"namespace"`
	OccurredAt string `json:"occurred_at"`
}

// Time off that is not paid under the demo policy. Everything else is paid leave
// and shows up as days, not as a pay adjustment.
var unpaidKinds = map[string]bool{"other": true}

type Ledger struct {
	mu        sync.RWMutex
	decided   map[int64]Event // latest decision per request; replay-safe
	recent    []Event
	consumed  int
	lastEvent time.Time
	dailyRate int
	period    string
}

func newLedger(dailyRateCents int, payPeriod string) *Ledger {
	return &Ledger{decided: map[int64]Event{}, dailyRate: dailyRateCents, period: payPeriod}
}

func (l *Ledger) Apply(ev Event) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.consumed++
	l.lastEvent = time.Now()
	if ev.Type == "timeoff.decided" {
		l.decided[ev.RequestID] = ev
	}
	l.recent = append(l.recent, ev)
	if len(l.recent) > 50 {
		l.recent = l.recent[len(l.recent)-50:]
	}
}

type EmployeeLedger struct {
	EmployeeID      int64          `json:"employee_id"`
	Employee        string         `json:"employee"`
	Team            string         `json:"team"`
	ApprovedDays    map[string]int `json:"approved_days"`
	PaidDays        int            `json:"paid_days"`
	UnpaidDays      int            `json:"unpaid_days"`
	AdjustmentCents int            `json:"adjustment_cents"`
	Approved        int            `json:"approved_requests"`
	Denied          int            `json:"denied_requests"`
}

type LedgerView struct {
	PayPeriod      string           `json:"pay_period"`
	PeriodStart    string           `json:"period_start"`
	PeriodEnd      string           `json:"period_end"`
	DailyRateCents int              `json:"daily_rate_cents"`
	Employees      []EmployeeLedger `json:"employees"`
	TotalAdjCents  int              `json:"total_adjustment_cents"`
}

func (l *Ledger) View(now time.Time) LedgerView {
	l.mu.RLock()
	defer l.mu.RUnlock()

	byEmp := map[int64]*EmployeeLedger{}
	for _, ev := range l.decided {
		e := byEmp[ev.EmployeeID]
		if e == nil {
			e = &EmployeeLedger{EmployeeID: ev.EmployeeID, Employee: ev.Employee, Team: ev.Team, ApprovedDays: map[string]int{}}
			byEmp[ev.EmployeeID] = e
		}
		switch ev.Status {
		case "approved":
			e.Approved++
			e.ApprovedDays[ev.Kind] += ev.Days
			if unpaidKinds[ev.Kind] {
				e.UnpaidDays += ev.Days
				e.AdjustmentCents -= ev.Days * l.dailyRate
			} else {
				e.PaidDays += ev.Days
			}
		case "denied":
			e.Denied++
		}
	}

	view := LedgerView{PayPeriod: l.period, DailyRateCents: l.dailyRate}
	view.PeriodStart, view.PeriodEnd = periodBounds(l.period, now)
	for _, e := range byEmp {
		view.Employees = append(view.Employees, *e)
		view.TotalAdjCents += e.AdjustmentCents
	}
	sort.Slice(view.Employees, func(i, j int) bool { return view.Employees[i].Employee < view.Employees[j].Employee })
	if view.Employees == nil {
		view.Employees = []EmployeeLedger{}
	}
	return view
}

func (l *Ledger) Recent() []Event {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]Event, len(l.recent))
	for i, ev := range l.recent {
		out[len(l.recent)-1-i] = ev
	}
	return out
}

func (l *Ledger) Stats() (consumed int, last time.Time) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.consumed, l.lastEvent
}

func periodBounds(period string, now time.Time) (string, string) {
	const layout = "2006-01-02"
	y, m, d := now.Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	switch period {
	case "weekly":
		start := today.AddDate(0, 0, -int((today.Weekday()+6)%7))
		return start.Format(layout), start.AddDate(0, 0, 6).Format(layout)
	case "biweekly":
		start := today.AddDate(0, 0, -int((today.Weekday()+6)%7))
		if (start.YearDay()/7)%2 == 1 {
			start = start.AddDate(0, 0, -7)
		}
		return start.Format(layout), start.AddDate(0, 0, 13).Format(layout)
	case "semimonthly":
		if d <= 15 {
			return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC).Format(layout), time.Date(y, m, 15, 0, 0, 0, 0, time.UTC).Format(layout)
		}
		return time.Date(y, m, 16, 0, 0, 0, 0, time.UTC).Format(layout), time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Format(layout)
	default:
		return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC).Format(layout), time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Format(layout)
	}
}
