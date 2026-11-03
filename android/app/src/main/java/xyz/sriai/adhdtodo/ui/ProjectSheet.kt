package xyz.sriai.adhdtodo.ui

import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.data.ProjectEntry

private enum class ProjectStep { Menu, Rename, NewSub, NewMilestone, Delete }

/** The repository page of a project (`QingYunA/withdo`), null for a project with none. */
fun githubUrl(project: Project): String? =
    project.repoSlug.takeIf { it.isNotBlank() }?.let { "https://github.com/$it" }

/**
 * The long-press sheet of a project row: its name and `QingYunA/withdo · 6 tasks`, then Rename…, New
 * subproject…, New milestone…, Open on GitHub (only with a repository), a hairline, Archive (Unarchive
 * in the fold) and Delete… in blocked red, which asks with the desktop's words first.
 */
@Composable
fun ProjectSheet(
    entry: ProjectEntry,
    archived: Boolean,
    actions: ProjectActions,
    onOpenUrl: (String) -> Unit,
    onDismiss: () -> Unit,
) {
    val project = entry.project
    val scope = rememberCoroutineScope()
    var step by remember { mutableStateOf(ProjectStep.Menu) }
    val tasks = pluralStringResource(R.plurals.project_tasks, entry.total, entry.total)
    val meta = listOfNotNull(project.repoSlug.takeIf { it.isNotBlank() }, tasks).joinToString(" · ")
    val github = githubUrl(project)

    fun run(then: () -> Unit = onDismiss, action: suspend () -> Unit) {
        scope.launch {
            try {
                action()
            } catch (e: CancellationException) {
                throw e
            } catch (_: Exception) {
                // The page reads again on its own; a refused archive leaves the row as it was.
            }
            then()
        }
    }

    when (step) {
        ProjectStep.Menu -> SheetFrame(onDismiss) {
            SheetTitle(project.name, meta)
            SheetRow(stringResource(R.string.manage_rename_item), icon = "folder", onClick = { step = ProjectStep.Rename })
            SheetRow(stringResource(R.string.manage_new_subproject), icon = "plus", onClick = { step = ProjectStep.NewSub })
            SheetRow(stringResource(R.string.milestone_new), icon = "diamond-gem", onClick = { step = ProjectStep.NewMilestone })
            if (github != null) {
                SheetRow(stringResource(R.string.manage_open_github), icon = "github", onClick = {
                    onOpenUrl(github)
                    onDismiss()
                })
            }
            SheetRule()
            SheetRow(
                label = stringResource(if (archived) R.string.manage_unarchive else R.string.manage_archive),
                icon = "inbox",
                onClick = {
                    run {
                        if (archived) actions.unarchiveProject(project.id) else actions.archiveProject(project.id)
                    }
                },
            )
            SheetRow(stringResource(R.string.manage_delete), icon = "trash", danger = true, onClick = { step = ProjectStep.Delete })
        }
        ProjectStep.Rename -> PromptSheet(
            title = stringResource(R.string.manage_rename_title),
            placeholder = stringResource(R.string.manage_project_name),
            confirm = stringResource(R.string.manage_save),
            initial = project.name,
            onDismiss = onDismiss,
            onConfirm = { name, _ -> actions.renameProject(project.id, name) },
        )
        ProjectStep.NewSub -> PromptSheet(
            title = stringResource(R.string.manage_new_subproject_title),
            subtitle = stringResource(R.string.manage_subproject_under, project.name),
            placeholder = stringResource(R.string.manage_project_name),
            confirm = stringResource(R.string.manage_create),
            onDismiss = onDismiss,
            onConfirm = { name, _ -> actions.createProject(name, project.id) },
        )
        ProjectStep.NewMilestone -> PromptSheet(
            title = stringResource(R.string.milestone_new_title),
            placeholder = stringResource(R.string.milestone_placeholder),
            confirm = stringResource(R.string.action_done),
            withDue = true,
            onDismiss = onDismiss,
            onConfirm = { title, due -> actions.createMilestone(project.id, title, due?.toString().orEmpty()) },
        )
        ProjectStep.Delete -> DeleteDialog(
            title = stringResource(R.string.manage_delete_project_title, project.name),
            body = if (entry.total > 0) {
                pluralStringResource(R.plurals.manage_delete_project_body, entry.total, entry.total)
            } else {
                stringResource(R.string.manage_delete_project_empty)
            },
            onConfirm = { run { actions.deleteProject(project.id) } },
            onDismiss = onDismiss,
        )
    }
}

/** The plus of the Projects header: a name field and Create; the new project opens. */
@Composable
fun NewProjectSheet(actions: ProjectActions, onDismiss: () -> Unit) {
    PromptSheet(
        title = stringResource(R.string.manage_new_project),
        placeholder = stringResource(R.string.manage_project_name),
        confirm = stringResource(R.string.manage_create),
        onDismiss = onDismiss,
        onConfirm = { name, _ -> actions.createProject(name, 0) },
    )
}
