plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.plugin.compose")
    id("org.jetbrains.kotlin.plugin.serialization")
    id("io.github.takahirom.roborazzi")
}

android {
    namespace = "xyz.sriai.adhdtodo"
    compileSdk = 37

    defaultConfig {
        applicationId = "xyz.sriai.adhdtodo"
        minSdk = 26
        targetSdk = 35
        versionCode = 3
        versionName = "0.6.2"
    }
    // Release signing comes from the environment (scripts/android-release.sh reads the passwords
    // from the keychain); the keystore lives outside the repo. Without it a release build is
    // unsigned.
    val releaseStore = System.getenv("ADHD_RELEASE_STORE_FILE")
    signingConfigs {
        if (releaseStore != null) {
            create("release") {
                storeFile = file(releaseStore)
                storePassword = System.getenv("ADHD_RELEASE_STORE_PASSWORD")
                keyAlias = System.getenv("ADHD_RELEASE_KEY_ALIAS")
                keyPassword = System.getenv("ADHD_RELEASE_KEY_PASSWORD")
            }
        }
    }
    buildTypes {
        getByName("release") {
            // No shrinking yet: R8 needs keep rules for the gomobile classes and serialization.
            isMinifyEnabled = false
            if (releaseStore != null) {
                signingConfig = signingConfigs.getByName("release")
            }
        }
    }
    buildFeatures {
        compose = true
        buildConfig = true
    }
    sourceSets {
        // The pictures have one home in the repo: no copies under android/.
        getByName("main").assets.directories.add("../../app/theme/sprites/avatars")
        getByName("main").assets.directories.add("../../app/theme/pixel-icons")
    }
    testOptions {
        unitTests.isIncludeAndroidResources = true
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

dependencies {
    // The Go core, built by scripts/android-core.sh (not committed).
    implementation(files("../core/core.aar"))
    implementation(platform("androidx.compose:compose-bom:2026.09.00"))
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.foundation:foundation")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.activity:activity-compose:1.13.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.12.0")
    debugImplementation("androidx.compose.ui:ui-tooling")
    implementation("androidx.compose.ui:ui-tooling-preview")
    testImplementation("junit:junit:4.13.2")
    testImplementation("org.robolectric:robolectric:4.17")
    testImplementation("io.github.takahirom.roborazzi:roborazzi:1.76.0")
    testImplementation("io.github.takahirom.roborazzi:roborazzi-compose:1.76.0")
    testImplementation("androidx.compose.ui:ui-test-junit4")
    debugImplementation("androidx.compose.ui:ui-test-manifest")
}
