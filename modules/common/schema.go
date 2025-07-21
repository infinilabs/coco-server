package common

const SCHEMA_VER = "v2"

func GetSchemaSuffix() string {
	return "-" + SCHEMA_VER
}
