package xyz.sriai.adhdtodo.ui.theme

/**
 * The status glyphs of app/theme/glyph.go: 12 x 12 grids of filled cells, the shape carrying the
 * meaning and the color repeating it. The Kotlin grids are checked against glyph.go by
 * GlyphParityTest.
 */
object Glyphs {
    val disc = listOf(
        "....####....",
        "..########..",
        ".##########.",
        ".##########.",
        "############",
        "############",
        "############",
        "############",
        ".##########.",
        ".##########.",
        "..########..",
        "....####....",
    )

    val ring = listOf(
        "....####....",
        "..##....##..",
        ".#........#.",
        ".#........#.",
        "#..........#",
        "#..........#",
        "#..........#",
        "#..........#",
        ".#........#.",
        ".#........#.",
        "..##....##..",
        "....####....",
    )

    private val doingHalf = listOf(
        6 to 3, 7 to 3, 6 to 4, 7 to 4, 8 to 4, 6 to 5, 7 to 5, 8 to 5,
        6 to 6, 7 to 6, 8 to 6, 6 to 7, 7 to 7, 8 to 7, 6 to 8, 7 to 8,
    )
    private val reviewDot = listOf(
        5 to 4, 6 to 4, 4 to 5, 5 to 5, 6 to 5, 7 to 5,
        4 to 6, 5 to 6, 6 to 6, 7 to 6, 5 to 7, 6 to 7,
    )
    private val check = listOf(
        3 to 6, 3 to 7, 4 to 7, 4 to 8, 5 to 8, 5 to 9,
        6 to 7, 6 to 8, 7 to 6, 7 to 7, 8 to 5, 8 to 6, 9 to 4, 9 to 5,
    )
    private val bang = listOf(
        5 to 2, 6 to 2, 5 to 3, 6 to 3, 5 to 4, 6 to 4,
        5 to 5, 6 to 5, 5 to 6, 6 to 6, 5 to 8, 6 to 8, 5 to 9, 6 to 9,
    )
    private val cancelSlash = listOf(3 to 8, 4 to 7, 5 to 6, 6 to 5, 7 to 4, 8 to 3)

    /** Every third ring cell in reading order, which the backlog's dashed ring leaves out. */
    private val dashed: Set<Pair<Int, Int>> = buildSet {
        var i = 0
        for (y in ring.indices) {
            for (x in ring[y].indices) {
                if (ring[y][x] == '#') {
                    if (i % 3 == 2) add(x to y)
                    i++
                }
            }
        }
    }

    private val cache = HashMap<String, List<String>>()

    /** The 12 rows of a status: 'c' the status color, 'w' white, '.' empty. */
    fun rows(status: String): List<String> = cache.getOrPut(status) {
        when (status) {
            "backlog" -> grid(ring, emptyList(), 'c', dashed)
            "doing" -> grid(ring, doingHalf, 'c')
            "review" -> grid(ring, reviewDot, 'c')
            "blocked" -> grid(disc, bang, 'w')
            "done" -> grid(disc, check, 'w')
            "cancelled" -> grid(ring, cancelSlash, 'c')
            "todo" -> grid(ring, emptyList(), 'c')
            else -> grid(ring, emptyList(), 'c')
        }
    }

    private fun grid(
        base: List<String>,
        overlay: List<Pair<Int, Int>>,
        mark: Char,
        drop: Set<Pair<Int, Int>> = emptySet(),
    ): List<String> {
        val cells = Array(12) { CharArray(12) { '.' } }
        for (y in 0 until 12) {
            for (x in 0 until 12) {
                cells[y][x] = when {
                    base[y][x] != '#' -> base[y][x]
                    (x to y) in drop -> '.'
                    else -> 'c'
                }
            }
        }
        for ((x, y) in overlay) {
            cells[y][x] = mark
        }
        return cells.map { String(it) }
    }
}
