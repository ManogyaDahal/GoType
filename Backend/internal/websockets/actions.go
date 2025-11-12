package websockets

type Action string
const (
	ActionDelete Action = "delete"
	ActionJoin   Action = "join"
)

//Function tells if the action is valid
func IsValidAction(a Action) bool {
	switch a {
	case ActionJoin ,ActionDelete:
		return true
	default:
		return false
	}
}
