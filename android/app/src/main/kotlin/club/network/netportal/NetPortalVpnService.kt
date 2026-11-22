package club.network.netportal

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.net.VpnService
import android.os.Build
import android.os.ParcelFileDescriptor
import mobile.Mobile
import mobile.PlatformBridge
import org.json.JSONArray
import org.json.JSONObject

class NetPortalVpnService : VpnService() {

    companion object {
        const val ACTION_START = "club.network.netportal.action.START"
        const val ACTION_STOP = "club.network.netportal.action.STOP"
        const val EXTRA_NODE_ID = "node_id"
        private const val CHANNEL_ID = "netportal_vpn"
        private const val NOTIFICATION_ID = 1001
    }

    private var tunInterface: ParcelFileDescriptor? = null

    private val bridge = object : PlatformBridge {
        override fun protectFD(fd: Long): Boolean {
            return protect(fd.toInt())
        }

        override fun onEvent(eventJSON: String?) {
        }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_STOP -> {
                stopTunnel()
                stopSelf()
                return START_NOT_STICKY
            }
            ACTION_START, null -> {
                val nodeId = intent?.getStringExtra(EXTRA_NODE_ID)?.trim().orEmpty()
                if (nodeId.isEmpty()) {
                    stopSelf()
                    return START_NOT_STICKY
                }
                startForegroundNotification()
                startTunnel(nodeId)
            }
        }
        return START_STICKY
    }

    private fun startForegroundNotification() {
        val manager = getSystemService(NotificationManager::class.java)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                getString(R.string.vpn_notification_channel),
                NotificationManager.IMPORTANCE_LOW
            )
            manager.createNotificationChannel(channel)
        }
        val openIntent = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_IMMUTABLE
        )
        val notification = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            Notification.Builder(this, CHANNEL_ID)
        } else {
            @Suppress("DEPRECATION")
            Notification.Builder(this)
        }
            .setContentTitle(getString(R.string.vpn_notification_title))
            .setSmallIcon(android.R.drawable.ic_lock_lock)
            .setContentIntent(openIntent)
            .setOngoing(true)
            .build()

        startForeground(NOTIFICATION_ID, notification)
    }

    private fun startTunnel(nodeId: String) {
        try {
            Mobile.init(filesDir.absolutePath, bridge)
            val node = findNode(nodeId)
            val rawConfig = node?.optString("rawConfig", "").orEmpty()
            val mtu = node?.optInt("mtu", 1280)?.takeIf { it > 0 } ?: 1280

            val builder = Builder()
                .setSession("NetPortal")
                .setMtu(mtu)
                .addDisallowedApplication(packageName)

            val addresses = extractIniValues(rawConfig, "Address")
            var hasV4 = false
            var hasV6 = false
            for (cidr in addresses) {
                val parts = cidr.split("/")
                val ip = parts[0].trim()
                val prefix = parts.getOrNull(1)?.trim()?.toIntOrNull()
                if (ip.contains(":")) {
                    builder.addAddress(ip, prefix ?: 128)
                    hasV6 = true
                } else if (ip.isNotEmpty()) {
                    builder.addAddress(ip, prefix ?: 32)
                    hasV4 = true
                }
            }
            if (!hasV4 && !hasV6) {
                builder.addAddress("10.8.1.2", 32)
                hasV4 = true
            }
            if (hasV4) builder.addRoute("0.0.0.0", 0)
            if (hasV6) builder.addRoute("::", 0)

            val dnsServers = extractIniValues(rawConfig, "DNS")
            if (dnsServers.isEmpty()) {
                builder.addDnsServer("1.1.1.1")
            } else {
                for (dns in dnsServers) {
                    if (dns.isNotEmpty()) builder.addDnsServer(dns)
                }
            }

            val pfd = builder.establish() ?: return
            // Ownership of the detached file descriptor transfers to Go's tun.CreateTUNFromFile.
            val detachedFd = pfd.detachFd()
            tunInterface = null
            Mobile.startAWG(nodeId, detachedFd.toLong())
        } catch (_: Exception) {
            stopTunnel()
            stopSelf()
        }
    }

    private fun findNode(nodeId: String): JSONObject? {
        return try {
            val arr = JSONArray(Mobile.listNodesJSON())
            for (i in 0 until arr.length()) {
                val obj = arr.getJSONObject(i)
                if (obj.optString("id") == nodeId) return obj
            }
            null
        } catch (_: Exception) {
            null
        }
    }

    private fun extractIniValues(rawConfig: String, key: String): List<String> {
        if (rawConfig.isEmpty()) return emptyList()
        val result = mutableListOf<String>()
        for (line in rawConfig.lines()) {
            val trimmed = line.trim()
            if (trimmed.startsWith("#") || trimmed.startsWith(";")) continue
            val eq = trimmed.indexOf('=')
            if (eq <= 0) continue
            if (trimmed.substring(0, eq).trim().equals(key, ignoreCase = true)) {
                val rhs = trimmed.substring(eq + 1)
                rhs.split(",").map { it.trim() }.filterTo(result) { it.isNotEmpty() }
            }
        }
        return result
    }

    private fun stopTunnel() {
        try {
            Mobile.stop()
        } catch (_: Exception) {
        }
        try {
            tunInterface?.close()
        } catch (_: Exception) {
        }
        tunInterface = null
    }

    override fun onDestroy() {
        stopTunnel()
        super.onDestroy()
    }
}
