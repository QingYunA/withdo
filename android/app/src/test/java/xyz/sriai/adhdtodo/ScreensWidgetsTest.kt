package xyz.sriai.adhdtodo

import android.content.Context
import android.graphics.drawable.GradientDrawable
import android.view.View
import android.widget.FrameLayout
import androidx.activity.ComponentActivity
import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.DpSize
import androidx.compose.ui.unit.dp
import androidx.glance.appwidget.ExperimentalGlanceRemoteViewsApi
import androidx.glance.appwidget.GlanceRemoteViews
import com.github.takahirom.roborazzi.captureRoboImage
import kotlinx.coroutines.runBlocking
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.ParameterizedRobolectricTestRunner
import org.robolectric.Robolectric
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import xyz.sriai.adhdtodo.data.Item
import xyz.sriai.adhdtodo.data.MinePage
import xyz.sriai.adhdtodo.testing.FixedTime
import xyz.sriai.adhdtodo.testing.Golden
import xyz.sriai.adhdtodo.testing.screenQualifiers
import xyz.sriai.adhdtodo.testing.screenshotFile
import xyz.sriai.adhdtodo.ui.Words
import xyz.sriai.adhdtodo.widget.AgentsContent
import xyz.sriai.adhdtodo.widget.NoteContent
import xyz.sriai.adhdtodo.widget.TodayContent
import xyz.sriai.adhdtodo.widget.agentsData
import xyz.sriai.adhdtodo.widget.todayData

/**
 * The three home-screen widgets (v43, section 4) on a launcher-like backdrop. The widgets are
 * not redrawn with ordinary Compose: Glance composes their content into the same RemoteViews the
 * launcher receives, and the test applies those into a FrameLayout and captures it, so the
 * picture shows what the home screen inflates. The data is the golden Mine page (Today gets one
 * more row so the 4 x 2 card shows its three).
 */
@OptIn(ExperimentalGlanceRemoteViewsApi::class)
@RunWith(ParameterizedRobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [35])
class ScreensWidgetsTest(private val case: Case) {

    @Test
    fun screenshot() {
        RuntimeEnvironment.setQualifiers(screenQualifiers(case.night, case.zh))
        val activity = Robolectric.buildActivity(ComponentActivity::class.java).setup().get()
        val density = activity.resources.displayMetrics.density
        fun px(dp: Int) = (dp * density).toInt()

        val frame = FrameLayout(activity).apply { background = backdrop(case.night) }
        activity.setContentView(frame, FrameLayout.LayoutParams(px(HOME_WIDTH), px(HOME_HEIGHT)))

        val res = activity.resources
        val words = Words(res, FixedTime.clock, FixedTime.zone)
        val mine = Golden.load<MinePage>("mine.golden.json")
        val today = mine.copy(today = mine.today + Item(id = 9, status = "todo", title = "Call the landlord", dueAt = "2026-10-20T10:00:00Z"))

        place(activity, frame, Slot(16, 24, 380, 164)) { TodayContent(todayData(today, words, res)) }
        place(activity, frame, Slot(16, 212, 182, 182)) { AgentsContent(agentsData(mine, "", words, res)) }
        place(activity, frame, Slot(214, 212, 182, 182)) { NoteContent(res.getString(R.string.widget_note)) }
        appIcons(activity, frame, listOf(418, 498))

        frame.measure(
            View.MeasureSpec.makeMeasureSpec(px(HOME_WIDTH), View.MeasureSpec.EXACTLY),
            View.MeasureSpec.makeMeasureSpec(px(HOME_HEIGHT), View.MeasureSpec.EXACTLY),
        )
        frame.layout(0, 0, px(HOME_WIDTH), px(HOME_HEIGHT))
        frame.captureRoboImage(screenshotFile(case.name))
    }

    /** Composes [content] at the slot's size into RemoteViews, as the launcher would receive them. */
    private fun place(context: Context, frame: FrameLayout, slot: Slot, content: @Composable () -> Unit) {
        val density = context.resources.displayMetrics.density
        val views = runBlocking {
            GlanceRemoteViews().compose(context, DpSize(slot.width.dp, slot.height.dp), content = content).remoteViews
        }
        val view = views.apply(context, frame)
        frame.addView(
            view,
            FrameLayout.LayoutParams((slot.width * density).toInt(), (slot.height * density).toInt()).apply {
                leftMargin = (slot.left * density).toInt()
                topMargin = (slot.top * density).toInt()
            },
        )
    }

    data class Slot(val left: Int, val top: Int, val width: Int, val height: Int)

    data class Case(val name: String, val night: Boolean, val zh: Boolean)

    companion object {
        private const val HOME_WIDTH = 412
        private const val HOME_HEIGHT = 574

        @JvmStatic
        @ParameterizedRobolectricTestRunner.Parameters(name = "{0}")
        fun cases(): List<Case> = listOf(
            Case("widgets-light", night = false, zh = false),
            Case("widgets-dark", night = true, zh = false),
            Case("widgets-light-zh", night = false, zh = true),
            Case("widgets-dark-zh", night = true, zh = true),
        )
    }
}

/** The mockup's wallpaper: the same three stops, light and dark. */
private fun backdrop(night: Boolean) = GradientDrawable(
    GradientDrawable.Orientation.TL_BR,
    if (night) intArrayOf(0xFF1C1F4A.toInt(), 0xFF2B1F3D.toInt(), 0xFF3A2A22.toInt())
    else intArrayOf(0xFFC9CBF5.toInt(), 0xFFE9D8F0.toInt(), 0xFFF6E7D6.toInt()),
)

/** Rows of four faint rounded squares standing in for the launcher's apps. */
private fun appIcons(context: Context, frame: FrameLayout, tops: List<Int>) {
    val density = context.resources.displayMetrics.density
    for (top in tops) {
        for (i in 0 until 4) {
            val icon = View(context).apply {
                background = GradientDrawable().apply {
                    cornerRadius = 14 * density
                    setColor(0x38FFFFFF)
                }
            }
            frame.addView(
                icon,
                FrameLayout.LayoutParams((56 * density).toInt(), (56 * density).toInt()).apply {
                    leftMargin = ((16 + i * 100 + 22) * density).toInt()
                    topMargin = (top * density).toInt()
                },
            )
        }
    }
}
