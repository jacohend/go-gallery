package core

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	gfcore "github.com/gloflow/gloflow/go/gf_core"
)

//-------------------------------------------------------------
type Runtime struct {
	DB         *DB
	RuntimeSys *gfcore.Runtime_sys
}

type DB struct {
	MongoDB *mongo.Database
}

//-------------------------------------------------------------
func RuntimeGet(pMongoDBhostStr string,
	pMongoDBnameStr string) (*Runtime, *gfcore.Gf_error) {

	// RUNTIME_SYS
	runtimeSys := &gfcore.Runtime_sys{
		Service_name_str: "gallery",
	}

	// DB
	db, gErr := DBinit(pMongoDBhostStr, pMongoDBnameStr, runtimeSys)
	if gErr != nil {
		log.WithFields(log.Fields{
			"db_host": pMongoDBhostStr,
			"db_name": pMongoDBnameStr,
		}).Fatal("Error acquiring database connection")

		return nil, gErr
	}

	runtimeSys.Mongo_db = db.MongoDB

	// RUNTIME
	runtime := &Runtime{
		DB:         db, 
		RuntimeSys: runtimeSys,
	}

	return runtime, nil
}

//-------------------------------------------------------------
func DBinit(pMongoHostStr string,
	pMongoDBNamestr string,
	pRuntimeSys     *gfcore.Runtime_sys) (*DB, *gfcore.Gf_error) {

	mongoURLstr  := fmt.Sprintf("mongodb://%s", pMongoHostStr)
	log.WithFields(log.Fields{
		"host":    pMongoHostStr,
		"db_name": pMongoDBNamestr,
	}).Info("Mongo conn info")

	//-------------------------------------------------------------
	// GF_GET_DB
	GFgetDBfun := func() (*mongo.Database, *gfcore.Gf_error) {

		mongoDB, gErr := gfcore.Mongo__connect_new(mongoURLstr,
			pMongoDBNamestr,
			pRuntimeSys)
		if gErr != nil {
			return nil, gErr
		}
		log.Info("mongodb connected...")
		
		return mongoDB, nil
	}

	//-------------------------------------------------------------
	mongoDB, gErr := GFgetDBfun()
	if gErr != nil {
		return nil, gErr
	}

	db := &DB{
		MongoDB: mongoDB,
	}

	return db, nil
}

//-------------------------------------------------------------