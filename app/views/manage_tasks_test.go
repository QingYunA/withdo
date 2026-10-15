package views

func taskProject(app *App, id int) string {
	t, _ := app.Task(id)
	return app.projectName(t.ProjectID)
}
