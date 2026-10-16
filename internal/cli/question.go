package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
	"github.com/XKHoshizora/specmetry/internal/project"
	"github.com/XKHoshizora/specmetry/internal/proposal"
	"github.com/XKHoshizora/specmetry/internal/question"
	"github.com/XKHoshizora/specmetry/internal/secrets"
	"github.com/XKHoshizora/specmetry/internal/termtext"
)

// questionAnswerDoc is the question-answer.v1 input.
type questionAnswerDoc struct {
	Schema    string          `json:"schema"`
	Question  string          `json:"question"`
	Action    question.Action `json:"action"`
	Answer    string          `json:"answer,omitempty"`
	Rationale string          `json:"rationale,omitempty"`
}

// text is what closes the question: the answer, or the dismissal's
// rationale.
func (d questionAnswerDoc) text() string {
	if d.Action == question.Dismiss {
		return d.Rationale
	}
	return d.Answer
}

func newQuestionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "question",
		Short: "List, answer and dismiss the questions to clarify",
	}
	// An unknown question command, an argument or a flag that cannot be
	// used are refused without quoting them, as for every command
	// (secureCommandLine).
	cmd.AddCommand(newQuestionListCmd(), newQuestionCloseCmd(question.Answer), newQuestionCloseCmd(question.Dismiss))
	return cmd
}

func newQuestionListCmd() *cobra.Command {
	var all, asJSON bool
	cmd := &cobra.Command{
		Use:   "list [--all]",
		Short: "List the open questions to clarify, or with --all every question",
		Long: `List the questions to clarify that proposals opened, in key order: the open
ones, or with --all the answered and dismissed ones too, with the answer or
the reason for dismissing, who gave it and on which channel. A question is
about a requirement or criterion key, about an item of its proposal (the
item's ref) while that item is not accepted, or about the project. Only a
person answers or dismisses a question (question answer, question dismiss).`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			w, err := prepare(ctx)
			if err != nil {
				return err
			}
			defer w.close()
			if err := w.open(ctx); err != nil {
				return err
			}
			var doc question.ListDoc
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				doc, err = question.ListDocument(ctx, tx, p.ID, all)
				return err
			}); err != nil {
				return err
			}
			if asJSON {
				return writeJSON(cmd.OutOrStdout(), "question-list.v1", doc)
			}
			return printQuestions(cmd.OutOrStdout(), doc, all)
		}),
	}
	cmd.Flags().BoolVar(&all, "all", false, "also list the answered and dismissed questions")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the question-list.v1 document")
	return cmd
}

func printQuestions(w io.Writer, doc question.ListDoc, all bool) error {
	var b strings.Builder
	counts := map[question.Status]int{}
	for _, q := range doc.Questions {
		counts[q.Status]++
	}
	switch {
	case len(doc.Questions) == 0 && all:
		b.WriteString("No questions.\n")
	case len(doc.Questions) == 0:
		b.WriteString("No open questions.\n")
	case all:
		fmt.Fprintf(&b, "%d questions: %d open, %d answered, %d dismissed.\n", len(doc.Questions), counts[question.Open],
			counts[question.Answered], counts[question.Dismissed])
	default:
		fmt.Fprintf(&b, "%d open questions.\n", len(doc.Questions))
	}
	for _, q := range doc.Questions {
		fmt.Fprintf(&b, "\n%s  %s\n", q.Key, describeQuestion(q))
		fmt.Fprintf(&b, "  %s\n", indented(q.Text))
		if q.Context != nil {
			fmt.Fprintf(&b, "  context: %s\n", indented(*q.Context))
		}
		if a := q.Answer; a != nil {
			verb := "answered"
			if a.Action == question.Dismiss {
				verb = "dismissed"
			}
			fmt.Fprintf(&b, "  %s by %s (%s) at seq %d: %s\n", verb, a.By, a.Channel, a.Seq, indented(a.Text))
		}
	}
	if counts[question.Open] > 0 {
		b.WriteString("\nOnly a person answers or dismisses a question, at a terminal:\n" +
			"  specmetry question answer <key> -m <answer>\n  specmetry question dismiss <key> -m <reason>\n")
	}
	// The texts come from proposals and people: escaped where they reach
	// the terminal, all at once (codex-review-027 B1).
	_, err := io.WriteString(w, termtext.Escape(b.String()))
	return err
}

// describeQuestion is a question's status, whether it blocks, what it is
// about and where it comes from, on one line.
func describeQuestion(q question.Question) string {
	blocking := "not blocking"
	if q.Blocking {
		blocking = "blocking"
	}
	about := "about the project"
	if q.About != nil {
		about = "about " + *q.About
		if proposal.IsLocalRef(*q.About) {
			about = "about ref " + *q.About + " of its proposal"
		}
	}
	return fmt.Sprintf("%s, %s, %s (proposal %s, seq %d)", q.Status, blocking, about, q.ProposalID, q.CreatedSeq)
}

// indented is text whose further lines are indented under the first. It
// is escaped (termtext) where it is written; a CRLF, which the escaping
// turns into a line feed, keeps the indentation.
func indented(text string) string { return strings.ReplaceAll(text, "\n", "\n    ") }

func newQuestionCloseCmd(action question.Action) *cobra.Command {
	var text, file string
	var asJSON bool
	name := "question " + string(action)
	cmd := &cobra.Command{
		Use:   string(action) + " <question> -m <answer>",
		Short: "Answer an open question (a person only)",
		Long: `Answer an open question to clarify (see question list), once: an answered
question does not change. A person only, at a terminal, after confirming;
-m is the answer, or pass a question-answer.v1 document with --file. The
answer changes no baseline: a change it calls for goes through a proposal.`,
		Args: cobra.MaximumNArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			d := channel.FromContext(ctx)
			// Before anything given is read (PD-14).
			if err := d.RequireHuman(name); err != nil {
				return err
			}
			in, err := questionInput(cmd, action, args, text, file)
			if err != nil {
				return err
			}
			w, err := prepare(ctx)
			if err != nil {
				return err
			}
			defer w.close()
			if err := secrets.Reject(w.scanner.Fields(
				secrets.Field{Pointer: "/question", Value: in.Question},
				secrets.Field{Pointer: "/answer", Value: in.Answer},
				secrets.Field{Pointer: "/rationale", Value: in.Rationale})); err != nil {
				return err
			}
			actor, err := w.actor()
			if err != nil {
				return err
			}
			if err := w.open(ctx); err != nil {
				return err
			}
			// A question closed already is stale before anybody is asked.
			var q question.Question
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				q, err = question.RequireOpen(ctx, tx, p.ID, in.Question)
				return err
			}); err != nil {
				return err
			}
			// The summary holds the question's, the person's and the
			// configuration's texts: escaped for the terminal
			// (codex-review-027 B1).
			op := channel.Operation{Name: name, Summary: termtext.Lines(closeSummary(q, in, actor, d.Channel)), Confirm: q.Key}
			if err := channel.Guard(d, cmd.InOrStdin(), cmd.OutOrStdout(), op); err != nil {
				return err
			}
			// Close checks again that the question is open, under the write
			// lock: it may have been closed while the person confirmed.
			var seq int64
			if err := w.tx(ctx, func(tx *sql.Tx, p project.Registered) error {
				seq, err = question.Close(ctx, tx, p.ID, q.Key, question.Closing{Action: action, Text: in.text(), By: actor,
					Channel: string(d.Channel)})
				return err
			}); err != nil {
				return err
			}
			past := "Answered"
			if action == question.Dismiss {
				past = "Dismissed"
			}
			return printf(cmd.OutOrStdout(), "%s", termtext.Escape(fmt.Sprintf("%s %s (seq %d). The baseline is unchanged.\n", past, q.Key, seq)))
		}),
	}
	help := "the answer"
	if action == question.Dismiss {
		cmd.Use = "dismiss <question> -m <reason>"
		cmd.Short = "Dismiss an open question, with the reason (a person only)"
		cmd.Long = `Dismiss an open question to clarify (see question list) that needs no answer,
for example because the item it is about was rejected, once: a dismissed
question does not change. A person only, at a terminal, after confirming;
-m is the reason, or pass a question-answer.v1 document with --file. It
changes no baseline.`
		help = "why the question needs no answer"
	}
	cmd.Flags().StringVarP(&text, "message", "m", "", help)
	cmd.Flags().StringVar(&file, "file", "", "read a question-answer.v1 document from this file instead")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print errors as error.v1")
	return cmd
}

// questionInput builds the question-answer.v1 input from the argument and
// -m, or from the file, and validates it: an answer has its text, a
// dismissal its rationale, and a document's action must be the command's.
func questionInput(cmd *cobra.Command, action question.Action, args []string, text, file string) (questionAnswerDoc, error) {
	var raw []byte
	if file != "" {
		if len(args) > 0 || cmd.Flags().Changed("message") {
			return questionAnswerDoc{}, apperr.New(apperr.InvalidInput, "--file replaces the question and -m; give one or the other")
		}
		b, err := readSmallFile(file)
		if err != nil {
			return questionAnswerDoc{}, err
		}
		raw = b
	} else {
		what := "the answer"
		if action == question.Dismiss {
			what = "the reason"
		}
		if len(args) != 1 || !cmd.Flags().Changed("message") {
			return questionAnswerDoc{}, apperr.Newf(apperr.InvalidInput, "name the question and give %s with -m, or pass --file", what)
		}
		// An empty -m is kept, for the schema to refuse.
		m := map[string]any{"schema": "specmetry.question-answer/v1", "question": args[0], "action": action}
		if action == question.Dismiss {
			m["rationale"] = text
		} else {
			m["answer"] = text
		}
		b, err := json.Marshal(m)
		if err != nil {
			return questionAnswerDoc{}, apperr.Wrap(apperr.Internal, err, "cannot encode the input")
		}
		raw = b
	}
	val, err := validator()
	if err != nil {
		return questionAnswerDoc{}, apperr.Wrap(apperr.Internal, err, "cannot load the exchange schemas")
	}
	if err := val.ValidateJSON("question-answer.v1", raw); err != nil {
		return questionAnswerDoc{}, err
	}
	var in questionAnswerDoc
	if err := json.Unmarshal(raw, &in); err != nil {
		return questionAnswerDoc{}, apperr.New(apperr.InvalidInput, "the input is not valid")
	}
	if in.Action != action {
		return questionAnswerDoc{}, apperr.Newf(apperr.InvalidInput, "the document's action is not %s: run question %s", action, in.Action).
			WithDetails(apperr.Detail{Pointer: "/action", Problem: "must be " + string(action) + " for question " + string(action)})
	}
	return in, nil
}

// closeSummary is what a person confirms: the question, the answer or
// the reason, and what is recorded.
func closeSummary(q question.Question, in questionAnswerDoc, actor string, ch channel.Channel) []string {
	lines := []string{"question: " + q.Key + ", " + describeQuestion(q), "asks: " + indented(q.Text)}
	if q.Context != nil {
		lines = append(lines, "context: "+indented(*q.Context))
	}
	if in.Action == question.Dismiss {
		lines = append(lines, "dismissed because: "+indented(in.Rationale))
	} else {
		lines = append(lines, "answer: "+indented(in.Answer))
	}
	return append(lines,
		"recorded as "+actor+" on "+string(ch)+"; the question then does not change",
		"the baseline is unchanged: a change this calls for goes through a proposal")
}
