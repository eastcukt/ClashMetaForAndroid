## Clash Meta for Android

A Graphical user interface of [Clash.Meta](https://github.com/MetaCubeX/Clash.Meta) for Android

### Feature

Feature of [Clash.Meta](https://github.com/MetaCubeX/Clash.Meta)

[<img src="https://fdroid.gitlab.io/artwork/badge/get-it-on.png"
     alt="Get it on F-Droid"
     height="80">](https://f-droid.org/packages/com.github.metacubex.clash.meta/)

### GEO database updates

Both `geoip.dat` and `geoip.metadb`, together with `geosite.dat` and ASN data,
are bundled in the APK and extracted before profile validation. Profiles using
either GeoIP mode can therefore load GEO rules on a fresh installation without
an initial database download. App upgrades refresh older bundled database files,
including differently capitalized `GeoIP.dat` files left by the core.

Enable automatic GEO updates in the profile's top-level YAML:

```yaml
geo-auto-update: true
geo-update-interval: 24 # hours
```

CMFA uses the profile's `geox-url` download sources and updates installed
GeoSite and ASN databases, plus the installed GeoIP format selected by
`geodata-mode` (`geoip.dat` when true, MMDB when false). Uninstalled formats
are skipped. An overdue database is checked after the
configuration loads; otherwise the next check is scheduled from the oldest
selected database's modification time. After either success or failure, the
next attempt waits the configured interval while the service remains running.
Switching profiles replaces the scheduled task, and stopping the service or
setting `geo-auto-update: false` cancels future attempts. An in-flight core
download is allowed to finish before another configuration is applied.

This schedules the core's existing GEO update operations, including its file
validation and matcher-cache refresh. It does not update subscriptions or
change routing rules. A profile must load successfully before its task starts;
if the installed database lacks a category required by the profile, import a
compatible database first through Settings > Meta Features > Geo Files.

### Requirement

- Android 5.0+ (minimum)
- Android 7.0+ (recommend)
- `armeabi-v7a` , `arm64-v8a`, `x86` or `x86_64` Architecture

### Fork releases

This fork provides **Build Fork Release** in GitHub Actions. Run it manually,
or push a tag matching `fork-v*`, to build signed APKs for all supported ABIs
and publish a GitHub pre-release with SHA-256 checksums. It also runs the GEO
scheduler tests and verifies every APK signature before publishing.

The fork uses package ID `com.github.eastcukt.clash.meta`, so it can coexist
with official CMFA. Import your profiles into the fork after installation.
Subsequent fork builds use the same signing key and an increasing version code
and can upgrade that installation. Prefer the `arm64-v8a` APK for modern phones,
or `universal` for all supported architectures.

The workflow requires repository Actions secrets `FORK_KEYSTORE_BASE64`,
`FORK_STORE_PASSWORD`, `FORK_KEY_ALIAS`, and `FORK_KEY_PASSWORD`. Keep a private
backup of this signing key: it is required to publish compatible app updates.
The `FORK_` secrets are used only by the fork release workflow; the upstream
release workflows require their own signing configuration.

### Build

1. Update submodules

   ```bash
   git submodule update --init --recursive
   ```

2. Install **OpenJDK 11**, **Android SDK**, **CMake** and **Golang**

3. Create `local.properties` in project root with

   ```properties
   sdk.dir=/path/to/android-sdk
   ```

4. (Optional) Custom app package name. Add the following configuration to `local.properties`.

   ```properties
   # config your ownn applicationId, or it will be 'com.github.metacubex.clash'
   custom.application.id=com.my.compile.clash
   # remove application id suffix, or the applicaion id will be 'com.github.metacubex.clash.alpha'
   remove.suffix=true

5. Create `signing.properties` in project root with

   ```properties
   keystore.path=/path/to/keystore/file
   keystore.password=<key store password>
   key.alias=<key alias>
   key.password=<key password>
   ```

6. Build

   ```bash
   ./gradlew app:assembleAlphaRelease
   ```

### Automation

APP package name is `com.github.metacubex.clash.meta`

- Toggle Clash.Meta service status
  - Send intent to activity `com.github.kr328.clash.ExternalControlActivity` with action `com.github.metacubex.clash.meta.action.TOGGLE_CLASH`
- Start Clash.Meta service
  - Send intent to activity `com.github.kr328.clash.ExternalControlActivity` with action `com.github.metacubex.clash.meta.action.START_CLASH`
- Stop Clash.Meta service
  - Send intent to activity `com.github.kr328.clash.ExternalControlActivity` with action `com.github.metacubex.clash.meta.action.STOP_CLASH`
- Import a profile
  - URL Scheme `clash://install-config?url=<encoded URI>` or `clashmeta://install-config?url=<encoded URI>`

### Contribution and Project Maintenance

#### Meta Kernel

- CMFA uses the kernel from `android-real` branch under `MetaCubeX/Clash.Meta`, which is a merge of the main `Alpha` branch and `android-open`.
  - If you want to contribute to the kernel, make PRs to `Alpha` branch of the Meta kernel repository.
  - If you want to contribute Android-specific patches to the kernel, make PRs to  `android-open` branch of the Meta kernel repository.

#### Maintenance

- When `MetaCubeX/Clash.Meta` kernel is updated to a new version, the `Update Dependencies` actions in this repo will be triggered automatically.
  - It will pull the new version of the meta kernel, update all the golang dependencies, and create a PR without manual intervention.
  - If there is any compile error in PR, you need to fix it before merging. Alternatively, you may merge the PR directly.
- Manually triggering `Build Pre-Release` actions will compile and publish a `PreRelease` version.
- Manually triggering `Build Release` actions will compile, tag and publish a `Release` version.
  - You must fill the blank `Release Tag` with the tag you want to release in the format of `v1.2.3`.
  - `versionName` and `versionCode` in `build.gradle.kts` will be automatically bumped to the tag you filled above.
