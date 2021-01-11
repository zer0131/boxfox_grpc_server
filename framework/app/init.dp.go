package app

import (
	"context"
	"github.com/zer0131/toolbox/log"
	"gopkg.in/ini.v1"
	"strconv"
	"strings"
)

var ConfigVal = &Config{}

func LoadConfig(ctx context.Context, path string) error {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		IgnoreInlineComment: true,
	}, path)

	if err != nil {
		log.Errorf(ctx, "err=%s", err.Error())
		return err
	}
	ConfigVal.BaseVal.Port, err = cfg.Section("base").Key("port[int]").Int64()
	if err != nil {
		log.Errorf(ctx, "config [port[int]] err. err=%s", err.Error())
		return err
	}
	ConfigVal.BaseVal.LogLevel = cfg.Section("base").Key("log_level").String()
	ConfigVal.BaseVal.LogSize, err = cfg.Section("base").Key("log_size[int]").Int64()
	if err != nil {
		log.Errorf(ctx, "config [log_size[int]] err. err=%s", err.Error())
		return err
	}
	ConfigVal.BaseVal.Group = cfg.Section("base").Key("group").String()
	ConfigVal.BaseVal.Project = cfg.Section("base").Key("project").String()
	ConfigVal.BaseVal.DisableWebServer, err = cfg.Section("base").Key("disable_web_server[bool]").Bool()
	if err != nil {
		log.Errorf(ctx, "config [disable_web_server[bool]] err. err=%s", err.Error())
		return err
	}
	ConfigVal.BaseVal.Type = cfg.Section("base").Key("type").String()

	if err := InitProjectConfig(ctx, path); err != nil {
		return err
	}

	return nil

}

func InitProjectConfig(ctx context.Context, path string) error {

	cfg, err := ini.LoadSources(ini.LoadOptions{
		IgnoreInlineComment: true,
	}, path)

	if err != nil {
		log.Errorf(ctx, "err=%s", err.Error())
		return err
	}
	ConfigVal.BoxfoxGrpcServerVal.A, err = cfg.Section("boxfox_grpc_server").Key("a[int]").Int64()
	if err != nil {
		log.Errorf(ctx, "config [a[int]] err. err=%s", err.Error())
		return err
	}
	tB := cfg.Section("boxfox_grpc_server").Key("b[int_array]").String()
	tBIntArr := strings.Split(tB, ",")
	for _, v := range tBIntArr {
		vInt, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			log.Errorf(ctx, "err=%s", err.Error())
			return err
		}
		ConfigVal.BoxfoxGrpcServerVal.B = append(ConfigVal.BoxfoxGrpcServerVal.B, vInt)
	}
	ConfigVal.BoxfoxGrpcServerVal.C = cfg.Section("boxfox_grpc_server").Key("c[string]").String()
	tD := cfg.Section("boxfox_grpc_server").Key("d[string_array]").String()
	ConfigVal.BoxfoxGrpcServerVal.D = strings.Split(tD, ",")
	if err != nil {
		log.Errorf(ctx, "err=%s", err.Error())
		return err
	}
	ConfigVal.BoxfoxGrpcServerVal.E, err = cfg.Section("boxfox_grpc_server").Key("e[bool]").Bool()
	if err != nil {
		log.Errorf(ctx, "config [e[bool]] err. err=%s", err.Error())
		return err
	}

	return nil

}
