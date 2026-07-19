package domain

import "testing"

func TestRoleValidity(t *testing.T) {
	for _, r := range []Role{RoleSuperadmin, RoleManager, RoleCashier} {
		if !r.Valid() {
			t.Errorf("%s should be valid", r)
		}
	}
	if Role("owner").Valid() {
		t.Error("unknown role must be invalid")
	}
}

func TestCatalogPermissions(t *testing.T) {
	// Cashiers must NOT be able to manage the catalog (edit prices etc.).
	if RoleCashier.CanManageCatalog() {
		t.Error("cashier must not manage catalog")
	}
	if !RoleManager.CanManageCatalog() || !RoleSuperadmin.CanManageCatalog() {
		t.Error("manager/superadmin must manage catalog")
	}
	if RoleCashier.CanViewReports() {
		t.Error("cashier must not view reports")
	}
}
