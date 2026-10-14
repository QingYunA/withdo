package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/egoist/mygo"
)

// notifyReminder shows a system notification for a due reminder. The first one asks the user for
// permission and blocks until they answer, so it must not run on the main thread.
func notifyReminder(r Reminder) {
	if !mygo.NotificationsSupported() {
		log.Printf("notifications are not available (is the app packaged?)")
		return
	}
	title, body := r.text()
	err := mygo.NewNotification(mygo.NotificationOptions{ID: r.notificationID(), Title: title, Body: body}).Show()
	if errors.Is(err, mygo.ErrNotificationsDenied) {
		log.Printf("notifications are denied for ADHD Todo; allow them in System Settings")
	} else if err != nil {
		log.Printf("notification: %v", err)
	}
}

// openTaskFromNotification handles a click on a reminder, also one from an earlier run.
func (s *shell) openTaskFromNotification(id string) {
	if taskID, ok := taskIDFromNotification(id); ok {
		s.showMain(fmt.Sprintf("#/main/task/%d", taskID))
	}
}
