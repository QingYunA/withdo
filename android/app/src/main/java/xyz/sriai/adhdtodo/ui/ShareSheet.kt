package xyz.sriai.adhdtodo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalLocale
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import xyz.sriai.adhdtodo.R
import xyz.sriai.adhdtodo.data.Project
import xyz.sriai.adhdtodo.ui.theme.Avatar
import xyz.sriai.adhdtodo.ui.theme.LocalAppType
import xyz.sriai.adhdtodo.ui.theme.LocalTokens
import xyz.sriai.adhdtodo.ui.theme.PixelIcon
import java.time.LocalDate

/** The share sheet's pages: the form, and the Project and When pickers it opens. */
enum class SharePage { Main, Project, When }

/** What the share sheet holds: the title being edited, the link, and the person's picks. */
data class ShareForm(
    val title: TextFieldValue,
    val url: String?,
    val projectId: Long? = null,
    val picked: ShareWhen? = null,
    val page: SharePage = SharePage.Main,
    val adding: Boolean = false,
)

/**
 * Share into Withdo (v43 P2), over the other app: the assistant and `Add to Withdo`, the title
 * field (selected, keyboard up), the link as a chip, the Project and When rows, and a 48 dp accent
 * Add. [read] is what the composer's parser read from the title (null when nothing), which When
 * takes until the person picks.
 */
@Composable
fun ShareSheet(
    form: ShareForm,
    projects: List<Project>,
    read: String?,
    today: LocalDate,
    onChange: (ShareForm) -> Unit,
    onAdd: () -> Unit,
    onDismiss: () -> Unit,
) {
    SheetFrame(onDismiss = onDismiss) {
        when (form.page) {
            SharePage.Main -> ShareMain(form, projects, read, onChange, onAdd)
            SharePage.Project -> ShareProjects(form, projects, onChange)
            SharePage.When -> ShareDays(form, read, today, onChange)
        }
    }
}

@Composable
private fun ShareMain(
    form: ShareForm,
    projects: List<Project>,
    read: String?,
    onChange: (ShareForm) -> Unit,
    onAdd: () -> Unit,
) {
    val tokens = LocalTokens.current
    val type = LocalAppType.current
    val canAdd = form.title.text.isNotBlank() && !form.adding
    Row(
        Modifier.padding(start = 20.dp, end = 20.dp, top = 4.dp, bottom = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Avatar("assistant", size = 20.dp)
        Text(
            stringResource(R.string.share_title),
            style = type.bodyStrong.copy(fontSize = 16.sp, fontWeight = FontWeight.SemiBold),
            color = tokens.text,
        )
    }
    SheetField(
        value = form.title,
        onValueChange = { onChange(form.copy(title = it)) },
        placeholder = "",
        onDone = { if (canAdd) onAdd() },
    )
    form.url?.let { LinkChip(it) }
    SheetRule()
    SheetRow(
        label = stringResource(R.string.share_project),
        icon = "folder",
        hint = projects.firstOrNull { it.id == form.projectId }?.name ?: stringResource(R.string.title_inbox),
        chevron = true,
        onClick = { onChange(form.copy(page = SharePage.Project)) },
    )
    SheetRow(
        label = stringResource(R.string.share_when),
        icon = "calendar",
        hint = whenLabel(shareWhen(form.picked, read != null), read),
        chevron = true,
        onClick = { onChange(form.copy(page = SharePage.When)) },
    )
    Box(
        Modifier
            .padding(start = 16.dp, end = 16.dp, top = 12.dp)
            .fillMaxWidth()
            .height(48.dp)
            .clip(RoundedCornerShape(12.dp))
            .background(if (canAdd) tokens.accent else tokens.accent.copy(alpha = 0.4f))
            .clickable(enabled = canAdd, onClick = onAdd),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            stringResource(R.string.share_add),
            style = type.bodyStrong.copy(fontSize = 16.sp, fontWeight = FontWeight.SemiBold),
            color = tokens.onAccent,
        )
    }
}

/** The link under the field: the accent link mark and `github.com › issues/219` on `bg.hover`. */
@Composable
private fun LinkChip(url: String) {
    val tokens = LocalTokens.current
    Row(
        Modifier
            .padding(start = 16.dp, top = 6.dp)
            .background(tokens.bgHover, RoundedCornerShape(6.dp))
            .padding(horizontal = 8.dp, vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        PixelIcon("external-link", size = 14.dp, tint = tokens.accent)
        Text(
            linkLabel(url),
            style = LocalAppType.current.meta,
            color = tokens.textSecondary,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

/** Project: the Inbox, then every project with its folder color; the chosen one is ticked. */
@Composable
private fun ShareProjects(form: ShareForm, projects: List<Project>, onChange: (ShareForm) -> Unit) {
    val tokens = LocalTokens.current
    val back = { onChange(form.copy(page = SharePage.Main)) }
    PageHeader(stringResource(R.string.share_project), back)
    SheetRow(
        label = stringResource(R.string.title_inbox),
        icon = "inbox",
        checked = form.projectId == null,
        onClick = { onChange(form.copy(projectId = null, page = SharePage.Main)) },
    )
    for (project in projects) {
        SheetRow(
            label = project.name,
            leading = { PixelIcon("folder", size = 20.dp, tint = tokens.projectColor(project.name)) },
            checked = form.projectId == project.id,
            onClick = { onChange(form.copy(projectId = project.id, page = SharePage.Main)) },
        )
    }
}

/** When: what the title reads as (when it reads as a date), No date, then the ready-made days. */
@Composable
private fun ShareDays(form: ShareForm, read: String?, today: LocalDate, onChange: (ShareForm) -> Unit) {
    val current = shareWhen(form.picked, read != null)
    val pick = { choice: ShareWhen -> onChange(form.copy(picked = choice, page = SharePage.Main)) }
    PageHeader(stringResource(R.string.share_when), { onChange(form.copy(page = SharePage.Main)) })
    if (read != null) {
        SheetRow(label = read, icon = "bell", checked = current == ShareWhen.Read, onClick = { pick(ShareWhen.Read) })
    }
    SheetRow(
        label = stringResource(R.string.section_no_date),
        checked = current == ShareWhen.NoDate,
        onClick = { pick(ShareWhen.NoDate) },
    )
    val locale = LocalLocale.current.platformLocale
    for (option in dueOptions(today)) {
        SheetRow(
            label = dueKindLabel(option.kind),
            hint = dueRowDate(option.date, locale),
            checked = (current as? ShareWhen.Day)?.option == option,
            onClick = { pick(ShareWhen.Day(option)) },
        )
    }
}

/** The When row's value: the parser's reading, `No date`, or the day's name. */
@Composable
private fun whenLabel(choice: ShareWhen, read: String?): String = when (choice) {
    ShareWhen.Read -> read.orEmpty()
    ShareWhen.NoDate -> stringResource(R.string.section_no_date)
    is ShareWhen.Day -> dueKindLabel(choice.option.kind)
}
