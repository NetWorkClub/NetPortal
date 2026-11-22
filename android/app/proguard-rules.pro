# Keep gomobile and NetPortalBridge entry points
-keep class mobile.** { *; }
-keepclassmembers class club.network.netportal.MainActivity$JsBridge {
    @android.webkit.JavascriptInterface <methods>;
}
