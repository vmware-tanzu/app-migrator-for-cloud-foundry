## app-migrator import

Import Cloud Foundry applications

```
app-migrator import [flags]
```

### Examples

```
app-migrator import
app-migrator import --exclude-orgs='^system$",p-*'
app-migrator import --exclude-orgs='system,p-spring-cloud-services' --export-dir=/tmp
app-migrator import --include-orgs='org1,org2' --export-dir=/tmp
```

### Options

```
      --exclude-orgs strings   Any orgs matching the regex(es) specified will be excluded (default [system])
  -h, --help                   help for import
      --include-orgs strings   Only orgs matching the regex(es) specified will be included
```

### Options inherited from parent commands

```
      --debug               Enable debug logging
      --export-dir string   Directory where apps will be placed or read (default "export")
```

### SEE ALSO

* [app-migrator](app-migrator.md)	 - The app-migrator CLI is a tool for migrating apps from one TAS (Tanzu Application Service) to another
* [app-migrator import app](app-migrator_import_app.md)	 - Import app
* [app-migrator import org](app-migrator_import_org.md)	 - Import org
* [app-migrator import space](app-migrator_import_space.md)	 - Import space

###### Auto generated by spf13/cobra on 28-Jul-2022
