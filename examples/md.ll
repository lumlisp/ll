; markdown_decode — Pure Lum Lisp Markdown → HTML converter

(define (markdown_decode md)
  (define lines (string-split md "\n"))
  (define result (parse_blocks lines 1))
  (string-join (car result) "\n"))

; Missing standard library functions
(define (cadr x) (car (cdr x)))

(define (string_find s needle)
  (define s_len (string-length s))
  (define n_len (string-length needle))
  (define found -1)
  (define i 0)
  (while (and (< i s_len) (= found -1))
    (if (and (<= (+ i n_len) s_len)
             (string=? (substring s i (+ i n_len)) needle))
      (set! found i)
      (set! i (+ i 1))))
  found)

; ============================================================
; Block-level parser
; ============================================================

(define (parse_blocks lines line_num)
  (if (null? lines) (list () line_num)
    (begin
      (define line (car lines))
      (define trimmed (string_trim_left line))
      (define rest (cdr lines))
      (cond
        ((= (string-length trimmed) 0)
          (parse_blocks rest (+ line_num 1)))
        ((string_prefix? trimmed "```")
          (parse_fenced_code lines line_num))
        ((string_prefix? trimmed "#")
          (parse_header lines line_num))
        ((and (>= (string-length trimmed) 3)
              (is_hr? trimmed))
          (begin
            (define tail (parse_blocks rest (+ line_num 1)))
            (list (cons "<hr>" (car tail)) (cadr tail))))
        ((string_prefix? trimmed ">")
          (parse_blockquote lines line_num))
        ((or (string_prefix? trimmed "- ")
             (string_prefix? trimmed "* ")
             (string_prefix? trimmed "+ "))
          (parse_unordered_list lines line_num))
        ((string_prefix? trimmed "1. ")
          (parse_ordered_list lines line_num))
        ((string_prefix? trimmed "|")
          (parse_table lines line_num))
        (else (parse_paragraph lines line_num))))))

(define (is_hr? line)
  (define text (string-trim line))
  (and (>= (string-length text) 3)
       (hr_chars? text (substring text 0 1) 0)))

(define (hr_chars? text first i)
  (if (>= i (string-length text)) (>= i 3)
    (if (string=? (substring text i (+ i 1)) first)
      (hr_chars? text first (+ i 1))
      #f)))

; ============================================================
; Header
; ============================================================

(define (parse_header lines line_num)
  (define line (string-trim (car lines)))
  (define level 0)
  (define pos 0)
  (define len (string-length line))
  (while (and (< pos len) (string=? (substring line pos (+ pos 1)) "#"))
    (begin (set! level (+ level 1)) (set! pos (+ pos 1))))
  (if (> level 6) (set! level 6))
  (define content (string-trim (substring line pos len)))
  (define tag (number->string level))
  (define html (string-append "<h" tag ">" (parse_inline content) "</h" tag ">"))
  (define tail (parse_blocks (cdr lines) (+ line_num 1)))
  (list (cons html (car tail)) (cadr tail)))

; ============================================================
; Fenced code block
; ============================================================

(define (parse_fenced_code lines line_num)
  (define open_line (car lines))
  (define trimmed (string-trim open_line))
  (define fence_char (substring trimmed 0 1))
  (define fence_len 0)
  (while (and (< fence_len (string-length trimmed))
              (string=? (substring trimmed fence_len (+ fence_len 1)) fence_char))
    (set! fence_len (+ fence_len 1)))
  (define lang (string-trim (substring trimmed fence_len (string-length trimmed))))
  (define result (collect_fenced_code (cdr lines) fence_char fence_len ""))
  (define code (car result))
  (define remaining (cadr result))
  (define escaped (escape_html code))
  (define class (if (> (string-length lang) 0)
    (string-append " class=\"language-" lang "\"") ""))
  (define html (string-append "<pre><code" class ">" escaped "</code></pre>"))
  (define tail (parse_blocks remaining (+ line_num 1)))
  (list (cons html (car tail)) (cadr tail)))

(define (collect_fenced_code lines fence_char fence_len acc)
  (if (null? lines) (list acc ())
    (begin
      (define line (car lines))
      (define trimmed (string_trim_left line))
      (if (detect_close_fence trimmed fence_char fence_len)
        (list acc (cdr lines))
        (collect_fenced_code (cdr lines) fence_char fence_len
          (string-append acc line "\n"))))))

(define (detect_close_fence line fence_char fence_len)
  (define len (string-length line))
  (if (< len fence_len) #f
    (begin
      (define i 0)
      (define ok #t)
      (while (and ok (< i len)
                   (string=? (substring line i (+ i 1)) fence_char))
        (set! i (+ i 1)))
      (>= i fence_len))))

; ============================================================
; Inline code
; ============================================================

(define (parse_inline text)
  (if (= (string-length text) 0) ""
    (inline_loop text 0 "")))

(define (inline_loop text pos acc)
  (if (>= pos (string-length text)) acc
    (begin
      (define ch (substring text pos (+ pos 1)))
      (define next (if (< (+ pos 1) (string-length text))
                     (substring text (+ pos 1) (+ pos 2)) ""))
      (cond
        ((string=? ch "`")
          (inline_code text (+ pos 1) acc))
        ((and (string=? ch "*") (string=? next "*"))
          (inline_bold text (+ pos 2) acc))
        ((string=? ch "*")
          (inline_italic text (+ pos 1) acc))
        ((and (string=? ch "_") (string=? next "_"))
          (inline_bold text (+ pos 2) acc))
        ((string=? ch "_")
          (inline_italic text (+ pos 1) acc))
        ((and (string=? ch "~") (string=? next "~"))
          (inline_strike text (+ pos 2) acc))
        ((and (string=? ch "!") (string=? next "["))
          (inline_image text (+ pos 2) acc))
        ((string=? ch "[")
          (inline_link text (+ pos 1) acc))
        ((and (string=? ch "\\") (< (+ pos 1) (string-length text)))
          (inline_loop text (+ pos 2)
            (string-append acc next)))
        (else
          (inline_loop text (+ pos 1)
            (string-append acc (html_esc ch))))))))

(define (inline_code text pos acc)
  (define end (find_from text pos "`"))
  (if (= end -1)
    (inline_loop text (+ pos 1) (string-append acc "`"))
    (inline_loop text (+ end 1)
      (string-append acc "<code>" (escape_html (substring text pos end)) "</code>"))))

(define (inline_bold text pos acc)
  (define end (find_closing text pos "**"))
  (if (= end -1)
    (inline_loop text pos (string-append acc "**"))
    (inline_loop text (+ end 2)
      (string-append acc "<strong>" (parse_inline (substring text pos end)) "</strong>"))))

(define (inline_italic text pos acc)
  (define end (find_closing text pos "*"))
  (if (= end -1)
    (inline_loop text pos (string-append acc "*"))
    (inline_loop text (+ end 1)
      (string-append acc "<em>" (parse_inline (substring text pos end)) "</em>"))))

(define (inline_strike text pos acc)
  (define end (find_closing text pos "~~"))
  (if (= end -1)
    (inline_loop text pos (string-append acc "~~"))
    (inline_loop text (+ end 2)
      (string-append acc "<del>" (parse_inline (substring text pos end)) "</del>"))))

; ============================================================
; Links
; ============================================================

(define (inline_link text pos acc)
  (define bracket_end (find_from text pos "]("))
  (if (= bracket_end -1)
    (inline_loop text pos (string-append acc "["))
    (begin
      (define link_text (substring text pos bracket_end))
      (define after_paren (+ bracket_end 2))
      (define paren_end (find_from text after_paren ")"))
      (if (= paren_end -1)
        (inline_loop text pos (string-append acc "[" link_text "]"))
        (begin
          (define raw_url (substring text after_paren paren_end))
          (define url_parts (split_at_space raw_url))
          (define url (car url_parts))
          (define title (if (null? (cdr url_parts)) ""
                          (string-append " title=\""
                            (string-trim (cadr url_parts)) "\"")))
          (inline_loop text (+ paren_end 1)
            (string-append acc "<a href=\"" (escape_html url) "\"" title ">"
              (parse_inline link_text) "</a>")))))))

(define (inline_image text pos acc)
  (define bracket_end (find_from text pos "]("))
  (if (= bracket_end -1)
    (inline_loop text pos (string-append acc "!["))
    (begin
      (define alt_text (substring text pos bracket_end))
      (define after_paren (+ bracket_end 2))
      (define paren_end (find_from text after_paren ")"))
      (if (= paren_end -1)
        (inline_loop text pos (string-append acc "![" alt_text "]"))
        (inline_loop text (+ paren_end 1)
          (string-append acc "<img src=\""
            (escape_html (substring text after_paren paren_end))
            "\" alt=\"" (escape_html alt_text) "\">"))))))

(define (split_at_space s)
  (define pos (string_find s " "))
  (if (= pos -1) (list s ())
    (list (substring s 0 pos) (substring s (+ pos 1) (string-length s)))))

; ============================================================
; Pattern finding helpers
; ============================================================

(define (find_closing text start pattern)
  (define len (string-length text))
  (define pat_len (string-length pattern))
  (define found -1)
  (define i start)
  (while (and (< i len) (= found -1))
    (if (and (<= (+ i pat_len) len)
             (string=? (substring text i (+ i pat_len)) pattern))
      (set! found i)
      (if (string=? (substring text i (+ i 1)) "\\")
        (set! i (+ i 2))
        (set! i (+ i 1)))))
  found)

(define (find_char text start char)
  (define len (string-length text))
  (define found -1)
  (define i start)
  (while (and (< i len) (= found -1))
    (if (string=? (substring text i (+ i 1)) char)
      (set! found i)
      (set! i (+ i 1))))
  found)

(define (find_from text start needle)
  (define len (string-length text))
  (define nlen (string-length needle))
  (define found -1)
  (define i start)
  (while (and (< i len) (= found -1))
    (if (and (<= (+ i nlen) len)
             (string=? (substring text i (+ i nlen)) needle))
      (set! found i)
      (set! i (+ i 1))))
  found)

; ============================================================
; Paragraph
; ============================================================

(define (parse_paragraph lines line_num)
  (define result (collect_paragraph lines ""))
  (define text (car result))
  (define remaining (cadr result))
  (if (> (string-length text) 0)
    (begin
      (define html (string-append "<p>" (parse_inline (string-trim text)) "</p>"))
      (define tail (parse_blocks remaining line_num))
      (list (cons html (car tail)) (cadr tail)))
    (parse_blocks remaining line_num)))

(define (collect_paragraph lines acc)
  (if (null? lines) (list acc ())
    (begin
      (define line (car lines))
      (define trimmed (string-trim line))
      (define rest (cdr lines))
      (if (= (string-length trimmed) 0)
        (list acc rest)
        (if (or (string_prefix? (string_trim_left line) "#")
                (string_prefix? (string_trim_left line) "```")
                (string_prefix? (string_trim_left line) ">")
                (string_prefix? (string_trim_left line) "|")
                (is_list_item? (string_trim_left line))
                (is_hr? trimmed))
          (list acc lines)
          (collect_paragraph rest (string-append acc line " ")))))))

(define (is_list_item? line)
  (or (string_prefix? line "- ")
      (string_prefix? line "* ")
      (string_prefix? line "+ ")
      (string_prefix? line "1. ")))

; ============================================================
; Lists
; ============================================================

(define (parse_unordered_list lines line_num)
  (define result (collect_list_items lines "- "))
  (define items (car result))
  (define remaining (cadr result))
  (define html (string-append "<ul>\n" (string-join items "\n") "\n</ul>"))
  (define tail (parse_blocks remaining line_num))
  (list (cons html (car tail)) (cadr tail)))

(define (parse_ordered_list lines line_num)
  (define result (collect_list_items lines "1. "))
  (define items (car result))
  (define remaining (cadr result))
  (define html (string-append "<ol>\n" (string-join items "\n") "\n</ol>"))
  (define tail (parse_blocks remaining line_num))
  (list (cons html (car tail)) (cadr tail)))

(define (collect_list_items lines prefix)
  (if (null? lines) (list () ())
    (begin
      (define items ())
      (define remaining lines)
      (define collecting #t)
      (while (and collecting (not (null? remaining)))
        (define line (car remaining))
        (define trimmed (string_trim_left line))
        (define trimmed_line (string-trim line))
        (if (= (string-length trimmed_line) 0)
          (set! collecting #f)
          (if (string_prefix? trimmed prefix)
            (begin
              (define content (parse_inline
                (substring trimmed (string-length prefix)
                  (string-length trimmed))))
              (set! items (append items
                (list (string-append "  <li>" content "</li>"))))
              (set! remaining (cdr remaining)))
            (set! collecting #f))))
      (list items remaining))))

; ============================================================
; Blockquote
; ============================================================

(define (parse_blockquote lines line_num)
  (define result (collect_blockquote lines ""))
  (define text (car result))
  (define remaining (cadr result))
  (define inner (markdown_decode text))
  (define html (string-append "<blockquote>\n" inner "\n</blockquote>"))
  (define tail (parse_blocks remaining line_num))
  (list (cons html (car tail)) (cadr tail)))

(define (collect_blockquote lines acc)
  (if (null? lines) (list acc ())
    (begin
      (define line (car lines))
      (define trimmed (string_trim_left line))
      (if (string_prefix? trimmed ">")
        (begin
          (define content (substring trimmed 1 (string-length trimmed)))
          (if (string_prefix? content " ")
            (set! content (substring content 1 (string-length content))))
          (collect_blockquote (cdr lines)
            (string-append acc content "\n")))
        (list acc lines)))))

; ============================================================
; Tables (pipe tables)
; ============================================================

(define (parse_table lines line_num)
  (define result (collect_table lines))
  (define html (table_to_html (car result)))
  (define tail (parse_blocks (cadr result) line_num))
  (list (cons html (car tail)) (cadr tail)))

(define (collect_table lines)
  (if (null? lines) (list () ())
    (begin
      (define rows ())
      (define remaining lines)
      (define collecting #t)
      (while (and collecting (not (null? remaining)))
        (define line (car remaining))
        (define trimmed (string-trim line))
        (if (string_prefix? trimmed "|")
          (begin
            (set! rows (append rows (list trimmed)))
            (set! remaining (cdr remaining)))
          (set! collecting #f)))
      (list rows remaining))))

(define (table_to_html rows)
  (if (null? rows) ""
    (begin
      (define has_header (and (> (length rows) 1)
        (string_contains? (substring (cadr rows) 0 2) "-")))
      (define header_cells (if has_header (parse_table_row (car rows)) ""))
      (define body_rows (if has_header (drop rows 2) (cdr rows)))
      (define thead (if has_header
        (string-append "    <thead>\n      <tr>" header_cells "</tr>\n    </thead>\n")
        ""))
      (define tbody (if (null? body_rows) ""
        (string-append "    <tbody>\n"
          (string-join (map (lambda (r)
            (string-append "      <tr>" (parse_table_row r) "</tr>")) body_rows) "\n")
          "\n    </tbody>\n")))
      (string-append "<table>\n" thead tbody "</table>"))))

(define (parse_table_row row)
  (define trimmed (string-trim row))
  (define inner (if (and (> (string-length trimmed) 0)
                         (string=? (substring trimmed 0 1) "|"))
                  (substring trimmed 1 (string-length trimmed)) trimmed))
  (define inner2 (if (and (> (string-length inner) 0)
                          (string=? (substring inner (- (string-length inner) 1)
                                    (string-length inner)) "|"))
                   (substring inner 0 (- (string-length inner) 1)) inner))
  (define cells (string-split inner2 "|"))
  (string-join (map (lambda (c) (string-append "<td>" (parse_inline (string-trim c)) "</td>"))
    cells) ""))

(define (string_contains? s needle) (> (string_find s needle) -1))

; ============================================================
; HTML escaping
; ============================================================

(define (escape_html text)
  (define result (string_replace text "&" "&amp;"))
  (set! result (string_replace result "<" "&lt;"))
  (set! result (string_replace result ">" "&gt;"))
  (set! result (string_replace result "\"" "&quot;"))
  result)

(define (html_esc ch)
  (cond
    ((string=? ch "&") "&amp;")
    ((string=? ch "<") "&lt;")
    ((string=? ch ">") "&gt;")
    ((string=? ch "\"") "&quot;")
    (else ch)))

; ============================================================
; String helpers
; ============================================================

(define (string_prefix? s prefix)
  (define len (string-length prefix))
  (and (>= (string-length s) len)
       (string=? (substring s 0 len) prefix)))

(define (string_trim_left s)
  (define len (string-length s))
  (define i 0)
  (while (and (< i len) (or (string=? (substring s i (+ i 1)) " ")
                            (string=? (substring s i (+ i 1)) "\t")))
    (set! i (+ i 1)))
  (if (= i 0) s (substring s i len)))

(define (string_replace s old new)
  (string-join (string-split s old) new))




(println (markdown_decode "# header"))
