package config

type Config struct {
	SeedFile       string `env:"JOT_SEED_FILE,required"`
	MasterPassword string `env:"JOT_MASTER_PASSWORD,required"`
	DataDir        string `env:"JOT_DATA_DIR,required"`
	BindAddr       string `env:"JOT_BINDADDR,default=localhost:8095"`
	Host           string `env:"JOT_HOST"`
}
