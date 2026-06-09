package config

type Environment string

const (
	Development Environment = "development"
	Production  Environment = "production"
)

type AppConfig struct {
	serverPort string
	db         DBConfig
	maxConn    uint
	env        Environment
	tls        bool
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func (c *AppConfig) ServerPort() string { return c.serverPort }
func (c *AppConfig) DB() DBConfig       { return c.db }
func (c *AppConfig) MaxConn() uint      { return c.maxConn }
func (c *AppConfig) Env() Environment   { return c.env }
func (c *AppConfig) TLS() bool          { return c.tls }

func New(opts ...func(*AppConfig)) *AppConfig {
	cfg := &AppConfig{
		serverPort: "3000",
		db:         DBConfig{SSLMode: "disable"},
		maxConn:    5,
		env:        Development,
	}
	for _, o := range opts {
		o(cfg)
	}
	return cfg
}

func WithServerPort(port string) func(*AppConfig) {
	return func(c *AppConfig) {
		c.serverPort = port
	}
}

func WithDB(db DBConfig) func(*AppConfig) {
	return func(c *AppConfig) {
		c.db = db
	}
}

func WithMaxConn(n uint) func(*AppConfig) {
	return func(c *AppConfig) {
		c.maxConn = n
	}
}

func WithEnvironment(env Environment) func(*AppConfig) {
	return func(c *AppConfig) {
		c.env = env
	}
}

func WithTLS(enabled bool) func(*AppConfig) {
	return func(c *AppConfig) {
		c.tls = enabled
	}
}
