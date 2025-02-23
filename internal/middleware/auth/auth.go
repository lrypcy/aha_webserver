package auth
import "github.com/gofiber/contrib/casbin"

authz := casbin.New(casbin.Config{
	ModelFilePath: "path/to/rbac_model.conf",
	PolicyAdapter: xormadapter.NewAdapter("mysql", "root:@tcp(127.0.0.1:3306)/"),
	Lookup: func(c *fiber.Ctx) string {
		// fetch authenticated user subject
	},
})

func AuthZ() *???{
	return authz
}