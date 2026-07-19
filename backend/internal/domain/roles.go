package domain

type Role string

const (
	RoleSuperadmin Role = "superadmin"
	RoleManager    Role = "manager"
	RoleCashier    Role = "cashier"
)

func (r Role) Valid() bool {
	switch r {
	case RoleSuperadmin, RoleManager, RoleCashier:
		return true
	}
	return false
}

// CanManageCatalog: create/edit/delete products, receive stock, edit prices.
// Cashiers cannot — enforced at the backend, not just hidden in the UI.
func (r Role) CanManageCatalog() bool {
	return r == RoleSuperadmin || r == RoleManager
}

// CanViewReports: dashboard profit/margin figures.
func (r Role) CanViewReports() bool {
	return r == RoleSuperadmin || r == RoleManager
}

// CanManageUsers: create staff accounts.
func (r Role) CanManageUsers() bool {
	return r == RoleSuperadmin || r == RoleManager
}
