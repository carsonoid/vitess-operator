package scripts

import (
	"bytes"
	"fmt"
	"text/template"

	// corev1 "k8s.io/api/core/v1"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
)

type ContainerScriptGenerator struct {
	ContainerType string
	Tablet        *vitessv1alpha2.VitessTablet
	Init          string
	Start         string
	PreStop       string
}

func NewContainerScriptGenerator(containerType string, tablet *vitessv1alpha2.VitessTablet) *ContainerScriptGenerator {
	return &ContainerScriptGenerator{
		ContainerType: containerType,
		Tablet:        tablet,
	}
}

func (csg *ContainerScriptGenerator) Generate() error {
	var err error
	switch csg.ContainerType {
	case "vttablet":
		csg.Init, err = csg.getTemplatedScript("vttabletinit", VTTabletInitTemplate)
		if err != nil {
			return err
		}
		csg.Start, err = csg.getTemplatedScript("vttabletstart", VTTabletStartTemplate)
		if err != nil {
			return err
		}
		csg.PreStop, err = csg.getTemplatedScript("vttabletPreStop", VTTabletPreStopTemplate)
		if err != nil {
			return err
		}
	case "mysql":
		csg.Init, err = csg.getTemplatedScript("mysqlinit", MySQLInitTemplate)
		if err != nil {
			return err
		}
		csg.Start, err = csg.getTemplatedScript("mysqlstart", MySQLStartTemplate)
		if err != nil {
			return err
		}
		csg.PreStop, err = csg.getTemplatedScript("mysqlPreStop", MySQLPreStopTemplate)
		if err != nil {
			return err
		}
	case "init_shard_master":
		csg.Start, err = csg.getTemplatedScript("init_shard_master", InitShardMaster)
		if err != nil {
			return err
		}
	case "vtctld":
		csg.Start, err = csg.getTemplatedScript("vtctld", VtCtldStart)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unsupported container type: %s", csg.ContainerType)
	}

	return nil
}

func (csg *ContainerScriptGenerator) getTemplatedScript(name string, templateStr string) (string, error) {
	tmpl, err := template.New(name).Parse(templateStr)
	if err != nil {
		return "", err
	}

	// For simplicity, the tablet and all parent objects are passed to the template.
	// This is safe while the templates are hard-coded. But if templates are ever made
	// end-user configurable could would potentially expose too much data and would need to be sanitized
	params := struct {
		Lockserver *vitessv1alpha2.VitessLockserver
		Cluster    *vitessv1alpha2.VitessCluster
		Cell       *vitessv1alpha2.VitessCell
		Keyspace   *vitessv1alpha2.VitessKeyspace
		Shard      *vitessv1alpha2.VitessShard
		Tablet     *vitessv1alpha2.VitessTablet
		ScopedName string
	}{
		csg.Tablet.GetLockserver(),
		csg.Tablet.GetCluster(),
		csg.Tablet.GetCell(),
		csg.Tablet.GetKeyspace(),
		csg.Tablet.GetShard(),
		csg.Tablet,
		csg.Tablet.GetScopedName(),
	}

	var out bytes.Buffer
	err = tmpl.Execute(&out, params)
	if err != nil {
		return "", err
	}

	return out.String(), nil
}
