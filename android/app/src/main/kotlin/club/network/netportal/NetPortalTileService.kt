package club.network.netportal

import android.content.Intent
import android.os.Build
import android.service.quicksettings.Tile
import android.service.quicksettings.TileService
import mobile.Mobile
import org.json.JSONObject

class NetPortalTileService : TileService() {

    override fun onStartListening() {
        super.onStartListening()
        refreshTileState()
    }

    override fun onClick() {
        super.onClick()
        val active = isTunnelActive()
        if (active) {
            val stopIntent = Intent(this, NetPortalVpnService::class.java).apply {
                action = NetPortalVpnService.ACTION_STOP
            }
            startService(stopIntent)
        } else {
            val startIntent = Intent(this, NetPortalVpnService::class.java).apply {
                action = NetPortalVpnService.ACTION_START
            }
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                startForegroundService(startIntent)
            } else {
                startService(startIntent)
            }
        }
        refreshTileState()
    }

    private fun isTunnelActive(): Boolean {
        return try {
            val json = JSONObject(Mobile.statusJSON())
            json.optString("status") == "connected"
        } catch (_: Exception) {
            false
        }
    }

    private fun refreshTileState() {
        val tile = qsTile ?: return
        tile.state = if (isTunnelActive()) Tile.STATE_ACTIVE else Tile.STATE_INACTIVE
        tile.updateTile()
    }
}
