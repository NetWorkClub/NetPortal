package club.network.netportal

import android.annotation.SuppressLint
import android.content.Intent
import android.net.VpnService
import android.os.Build
import android.os.Bundle
import android.webkit.JavascriptInterface
import android.webkit.WebResourceRequest
import android.webkit.WebResourceResponse
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.webkit.WebViewAssetLoader
import mobile.Mobile
import org.json.JSONObject

class MainActivity : AppCompatActivity() {

    private var pendingNodeId: String = ""

    private val vpnConsentLauncher = registerForActivityResult(
        ActivityResultContracts.StartActivityForResult()
    ) { result ->
        if (result.resultCode == RESULT_OK) {
            launchVpnService(pendingNodeId)
        }
    }

    @SuppressLint("SetJavaScriptEnabled")
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        try {
            Mobile.init(filesDir.absolutePath, null)
        } catch (_: Exception) {
        }

        val assetLoader = WebViewAssetLoader.Builder()
            .addPathHandler("/assets/", WebViewAssetLoader.AssetsPathHandler(this))
            .build()

        val webView = WebView(this)
        webView.settings.javaScriptEnabled = true
        webView.settings.domStorageEnabled = true
        webView.webViewClient = object : WebViewClient() {
            override fun shouldInterceptRequest(
                view: WebView,
                request: WebResourceRequest
            ): WebResourceResponse? {
                return assetLoader.shouldInterceptRequest(request.url)
            }
        }
        webView.addJavascriptInterface(JsBridge(), "NetPortalBridge")
        setContentView(webView)
        webView.loadUrl("https://appassets.androidplatform.net/assets/index.html")
    }

    private fun startVpnWithConsent(nodeId: String) {
        pendingNodeId = nodeId
        val prepareIntent = VpnService.prepare(this)
        if (prepareIntent != null) {
            vpnConsentLauncher.launch(prepareIntent)
        } else {
            launchVpnService(nodeId)
        }
    }

    private fun launchVpnService(nodeId: String) {
        val intent = Intent(this, NetPortalVpnService::class.java).apply {
            action = NetPortalVpnService.ACTION_START
            putExtra(NetPortalVpnService.EXTRA_NODE_ID, nodeId)
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            startForegroundService(intent)
        } else {
            startService(intent)
        }
    }

    private fun stopVpnService() {
        val intent = Intent(this, NetPortalVpnService::class.java).apply {
            action = NetPortalVpnService.ACTION_STOP
        }
        startService(intent)
    }

    inner class JsBridge {
        @JavascriptInterface
        fun connect(payloadJson: String) {
            val obj = JSONObject(payloadJson)
            val nodeId = obj.optString("nodeId", obj.optString("targetName", ""))
            runOnUiThread { startVpnWithConsent(nodeId) }
        }

        @JavascriptInterface
        fun disconnect() {
            runOnUiThread { stopVpnService() }
        }

        @JavascriptInterface
        fun probeNode(nodeJson: String, kind: String): String {
            return try {
                Mobile.probeNodeJSON(nodeJson, kind)
            } catch (e: Exception) {
                JSONObject()
                    .put("nodeId", "")
                    .put("tcpMs", -1)
                    .put("antiBot", "Error")
                    .put("error", e.message ?: "probe failed")
                    .toString()
            }
        }

        @JavascriptInterface
        fun importConfig(name: String, content: String): String {
            return Mobile.importConfig(name, content)
        }

        @JavascriptInterface
        fun listNodes(): String {
            return Mobile.listNodesJSON()
        }

        @JavascriptInterface
        fun status(): String {
            return Mobile.statusJSON()
        }
    }
}
