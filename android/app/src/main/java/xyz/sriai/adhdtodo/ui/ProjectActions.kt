package xyz.sriai.adhdtodo.ui

import android.content.Context
import xyz.sriai.adhdtodo.data.MilestonePatch

/**
 * What the Projects tab, the project page and the milestone sheets ask the core to do. Every call
 * suspends off the UI thread in App and throws the core's sentence when it fails, which the sheet
 * that asked shows under its field. The defaults do nothing, for the screenshots.
 */
data class ProjectActions(
    /** Makes a project, a sub-project of [parentId] unless that is 0. */
    val createProject: suspend (name: String, parentId: Long) -> Unit = { _, _ -> },
    val renameProject: suspend (id: Long, name: String) -> Unit = { _, _ -> },
    val archiveProject: suspend (id: Long) -> Unit = {},
    val unarchiveProject: suspend (id: Long) -> Unit = {},
    val deleteProject: suspend (id: Long) -> Unit = {},
    /** [dueAt] is `2026-11-14`, "" for none. */
    val createMilestone: suspend (projectId: Long, title: String, dueAt: String) -> Unit = { _, _, _ -> },
    val updateMilestone: suspend (id: Long, patch: MilestonePatch) -> Unit = { _, _ -> },
)

/** A project page shows its tasks as the two columns of a list or as a board. */
enum class ProjectView { List, Board }

/** The List | Board choice of each project, kept on this phone. */
object ProjectViewPrefs {
    private const val PREFS = "project_views"

    fun get(context: Context, projectId: Long): ProjectView =
        if (context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getBoolean("board_$projectId", false)) {
            ProjectView.Board
        } else {
            ProjectView.List
        }

    fun set(context: Context, projectId: Long, view: ProjectView) {
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit()
            .putBoolean("board_$projectId", view == ProjectView.Board)
            .apply()
    }
}
