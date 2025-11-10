env "local" {
  src = "file://db/migrations"
  url = "postgres://identity:identity_dev_password@localhost:5432/identity_db?sslmode=disable"
  dev = "docker://postgres/16/dev"
}

env "docker" {
  src = "file://db/migrations"
  url = "postgres://identity:identity_dev_password@postgres:5432/identity_db?sslmode=disable"
  dev = "docker://postgres/16/dev"
}
