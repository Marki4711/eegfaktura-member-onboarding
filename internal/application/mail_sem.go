package application

// mailSem limits concurrent outbound mail goroutines to prevent SMTP overload.
// Both ApplicationService (submission mails) and AdminApplicationService (approval
// mails) share this semaphore — at most 10 mails are in-flight at any time.
var mailSem = make(chan struct{}, 10)

func acquireMailSem() { mailSem <- struct{}{} }
func releaseMailSem() { <-mailSem }
