package model

// Role представляет уровень привилегий пользователя.
type Role string

const (
	RoleUser      Role = "user"
	RoleModerator Role = "moderator"
	RoleAdmin     Role = "admin"
	RoleRoot      Role = "root"
)

// Level возвращает числовой уровень роли для сравнения.
// Чем выше число, тем больше привилегий.
// Для неизвестной роли возвращает -1.
func (r Role) Level() int {
	switch r {
	case RoleUser:
		return 0
	case RoleModerator:
		return 1
	case RoleAdmin:
		return 2
	case RoleRoot:
		return 3
	default:
		return -1
	}
}

// CanActOn проверяет, может ли actor выполнять действия над target.
// actor может воздействовать только на аккаунты с меньшим уровнем привилегий.
func CanActOn(actor, target Role) bool {
	aLvl := actor.Level()
	tLvl := target.Level()
	if aLvl < 0 || tLvl < 0 {
		return false
	}
	return aLvl > tLvl
}

// MinRole проверяет, что роль пользователя не ниже требуемой.
func MinRole(userRole Role, required Role) bool {
	return userRole.Level() >= required.Level()
}