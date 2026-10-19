package data

import "time"

// Snapshot is everything the windows show, read together: the projects, the tasks outside archived
// projects with their reports, the milestones and the overview. Store.Snapshot reads it in a fixed
// number of queries, however many tasks there are.
type Snapshot struct {
	Projects   []Project // not archived, by name
	Archived   []Project
	Categories []Category
	Tasks      []Task // newest first
	Milestones map[int][]Milestone
	// Reports are the reports of each task, newest first; a task without any is not in the map.
	Reports  map[int][]Report
	Overview Overview
}

// LatestReport is the newest report an agent filed against a task, or nil. GitHub's own reports are
// not the agent speaking, so they do not count as its last word.
func (d TaskDetail) LatestReport() *Report { return d.newest(false) }

// GitHubReport is the newest report GitHub filed against a task, or nil.
func (d TaskDetail) GitHubReport() *Report { return d.newest(true) }

func (d TaskDetail) newest(github bool) *Report {
	for i := range d.Reports { // newest first
		if (d.Reports[i].Agent == GitHubAgent) == github {
			return &d.Reports[i]
		}
	}
	return nil
}

// Details are the tasks with their reports, by task id.
func (s Snapshot) Details() map[int]TaskDetail {
	out := make(map[int]TaskDetail, len(s.Tasks))
	for _, t := range s.Tasks {
		out[t.ID] = TaskDetail{Task: t, Reports: s.Reports[t.ID]}
	}
	return out
}

// LatestReport is TaskDetail.LatestReport of a task.
func (s Snapshot) LatestReport(id int) *Report {
	return TaskDetail{Reports: s.Reports[id]}.LatestReport()
}

// GitHubReport is TaskDetail.GitHubReport of a task.
func (s Snapshot) GitHubReport(id int) *Report {
	return TaskDetail{Reports: s.Reports[id]}.GitHubReport()
}

// ProjectName is the name of a task's project, "" for the Inbox or a project the snapshot does not
// list.
func (s Snapshot) ProjectName(id *int) string {
	if id == nil {
		return ""
	}
	for _, p := range s.Projects {
		if p.ID == *id {
			return p.Name
		}
	}
	return ""
}

// StaleHours is how long an active task may stay silent before it counts as stale.
func (s Snapshot) StaleHours() int {
	if s.Overview.StaleHours == 0 {
		return DefaultStaleHours
	}
	return s.Overview.StaleHours
}

// Describe is a task the way the overview has it: with the report an agent filed last and whether it
// has gone stale at now.
func (s Snapshot) Describe(t Task, now time.Time) OverviewTask {
	ot := OverviewTask{Task: t}
	since := t.UpdatedAt
	if r := s.LatestReport(t.ID); r != nil {
		ot.LatestReport = r
		since = r.CreatedAt
	}
	ot.HoursSinceActivity = int(now.Sub(since) / time.Hour)
	ot.Stale = ot.HoursSinceActivity >= s.StaleHours()
	return ot
}
