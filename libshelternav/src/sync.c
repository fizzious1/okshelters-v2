/**
 * sync.c -- SQLite database access for local shelter cache.
 *
 * Opens the database in WAL mode for concurrent read performance.
 * All queries use parameterized statements — no user-supplied SQL.
 */

#include "shelternav.h"
#include "sqlite3.h"

#include <string.h>

/* -------------------------------------------------------------------
 * Module-level state (single open database)
 * ------------------------------------------------------------------- */

static sqlite3 *g_db = NULL;

/* -------------------------------------------------------------------
 * sn_db_open
 * ------------------------------------------------------------------- */

int sn_db_open(const char *path)
{
    if (path == NULL) {
        return SN_ERR_INVALID_ARG;
    }
    if (g_db != NULL) {
        /* A database is already open.  Close it first. */
        sn_db_close();
    }

    int rc = sqlite3_open(path, &g_db);
    if (rc != SQLITE_OK) {
        g_db = NULL;
        return SN_ERR_DB_OPEN;
    }

    /* Enable WAL mode for concurrent reads. */
    char *err_msg = NULL;
    rc = sqlite3_exec(g_db, "PRAGMA journal_mode=WAL;", NULL, NULL, &err_msg);
    if (rc != SQLITE_OK) {
        sqlite3_free(err_msg);
        sqlite3_close(g_db);
        g_db = NULL;
        return SN_ERR_DB_OPEN;
    }

    return SN_OK;
}

/* -------------------------------------------------------------------
 * sn_db_close
 * ------------------------------------------------------------------- */

int sn_db_close(void)
{
    if (g_db == NULL) {
        return SN_OK; /* nothing to close */
    }

    int rc = sqlite3_close(g_db);
    g_db = NULL;

    return (rc == SQLITE_OK) ? SN_OK : SN_ERR_DB_CLOSE;
}

/* -------------------------------------------------------------------
 * sn_db_sync -- read all shelters into the KD-tree
 * ------------------------------------------------------------------- */

int sn_db_sync(SN_KDNode **tree)
{
    if (tree == NULL) {
        return SN_ERR_INVALID_ARG;
    }
    if (g_db == NULL) {
        return SN_ERR_DB_OPEN;
    }

    /* Parameterized query (no user-supplied SQL). */
    const char *sql =
        "SELECT id, lat, lon, status, capacity, name, address "
        "FROM shelters "
        "WHERE status != 0;";

    sqlite3_stmt *stmt = NULL;
    int rc = sqlite3_prepare_v2(g_db, sql, -1, &stmt, NULL);
    if (rc != SQLITE_OK) {
        return SN_ERR_DB_QUERY;
    }

    int synced = 0;

    while ((rc = sqlite3_step(stmt)) == SQLITE_ROW) {
        SN_Shelter shelter;
        memset(&shelter, 0, sizeof(shelter));

        shelter.id       = sqlite3_column_int(stmt, 0);
        shelter.lat      = sqlite3_column_double(stmt, 1);
        shelter.lon      = sqlite3_column_double(stmt, 2);
        shelter.status   = (uint8_t)sqlite3_column_int(stmt, 3);
        shelter.capacity = (uint16_t)sqlite3_column_int(stmt, 4);

        const char *name = (const char *)sqlite3_column_text(stmt, 5);
        if (name != NULL) {
            strncpy(shelter.name, name, sizeof(shelter.name) - 1);
            shelter.name[sizeof(shelter.name) - 1] = '\0';
        }

        const char *addr = (const char *)sqlite3_column_text(stmt, 6);
        if (addr != NULL) {
            strncpy(shelter.address, addr, sizeof(shelter.address) - 1);
            shelter.address[sizeof(shelter.address) - 1] = '\0';
        }

        int ins_rc = sn_kdtree_insert(tree, &shelter);
        if (ins_rc != SN_OK) {
            sqlite3_finalize(stmt);
            return ins_rc;
        }

        synced++;
    }

    sqlite3_finalize(stmt);

    if (rc != SQLITE_DONE) {
        return SN_ERR_DB_QUERY;
    }

    return synced;
}
