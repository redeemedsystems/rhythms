package domain

// User is one signed-in account. A row can exist before its owner has ever
// signed in — an admin invites an email, which creates a User with
// GoogleSub still empty; the first successful Google sign-in for that email
// fills it in (see UserRepo.SetGoogleSub). Signing in with an email that
// has no User row at all is rejected — there is no open signup.
type User struct {
	ID        int64
	Email     string
	GoogleSub string // "" until the invited email's first successful sign-in
	IsAdmin   bool
	CreatedAt string
}
