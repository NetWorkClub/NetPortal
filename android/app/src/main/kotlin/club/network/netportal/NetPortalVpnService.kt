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
            // Events can be forwarded to MainActivity broadcast or WebView if needed.
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
                val nodeId = intent?.getStringExtra(EXTRA_NODE_ID) ?: ""
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
            val builder = Builder()
                .setSession("NetPortal")
                .setMtu(1280)
                .addAddress("10.8.1.2", 32)
                .addRoute("0.0.0.0", 0)
                .addDnsServer("1.1.1.1")
                .addDisallowedApplication(packageName)

            val pfd = builder.establish() ?: return
            tunInterface = pfd
            if (nodeId.isNotEmpty()) {
                Mobile.startAWG(nodeId, pfd.detachFd().toLong())
            }
        } catch (_: Exception) {
            stopTunnel()
            stopSelf()
        }
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
