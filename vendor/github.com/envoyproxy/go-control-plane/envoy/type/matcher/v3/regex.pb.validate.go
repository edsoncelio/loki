// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: envoy/type/matcher/v3/regex.proto

package matcherv3

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
	_ = sort.Sort
)

// Validate checks the field values on RegexMatcher with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *RegexMatcher) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on RegexMatcher with the rules defined
// in the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in RegexMatcherMultiError, or
// nil if none found.
func (m *RegexMatcher) ValidateAll() error {
	return m.validate(true)
}

func (m *RegexMatcher) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if utf8.RuneCountInString(m.GetRegex()) < 1 {
		err := RegexMatcherValidationError{
			field:  "Regex",
			reason: "value length must be at least 1 runes",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	switch v := m.EngineType.(type) {
	case *RegexMatcher_GoogleRe2:
		if v == nil {
			err := RegexMatcherValidationError{
				field:  "EngineType",
				reason: "oneof value cannot be a typed-nil",
			}
			if !all {
				return err
			}
			errors = append(errors, err)
		}

		if all {
			switch v := interface{}(m.GetGoogleRe2()).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, RegexMatcherValidationError{
						field:  "GoogleRe2",
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, RegexMatcherValidationError{
						field:  "GoogleRe2",
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(m.GetGoogleRe2()).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return RegexMatcherValidationError{
					field:  "GoogleRe2",
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	default:
		_ = v // ensures v is used
	}

	if len(errors) > 0 {
		return RegexMatcherMultiError(errors)
	}

	return nil
}

// RegexMatcherMultiError is an error wrapping multiple validation errors
// returned by RegexMatcher.ValidateAll() if the designated constraints aren't met.
type RegexMatcherMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m RegexMatcherMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m RegexMatcherMultiError) AllErrors() []error { return m }

// RegexMatcherValidationError is the validation error returned by
// RegexMatcher.Validate if the designated constraints aren't met.
type RegexMatcherValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e RegexMatcherValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e RegexMatcherValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e RegexMatcherValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e RegexMatcherValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e RegexMatcherValidationError) ErrorName() string { return "RegexMatcherValidationError" }

// Error satisfies the builtin error interface
func (e RegexMatcherValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sRegexMatcher.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = RegexMatcherValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = RegexMatcherValidationError{}

// Validate checks the field values on RegexMatchAndSubstitute with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *RegexMatchAndSubstitute) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on RegexMatchAndSubstitute with the
// rules defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// RegexMatchAndSubstituteMultiError, or nil if none found.
func (m *RegexMatchAndSubstitute) ValidateAll() error {
	return m.validate(true)
}

func (m *RegexMatchAndSubstitute) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if m.GetPattern() == nil {
		err := RegexMatchAndSubstituteValidationError{
			field:  "Pattern",
			reason: "value is required",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if all {
		switch v := interface{}(m.GetPattern()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, RegexMatchAndSubstituteValidationError{
					field:  "Pattern",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, RegexMatchAndSubstituteValidationError{
					field:  "Pattern",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetPattern()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return RegexMatchAndSubstituteValidationError{
				field:  "Pattern",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if !_RegexMatchAndSubstitute_Substitution_Pattern.MatchString(m.GetSubstitution()) {
		err := RegexMatchAndSubstituteValidationError{
			field:  "Substitution",
			reason: "value does not match regex pattern \"^[^\\x00\\n\\r]*$\"",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return RegexMatchAndSubstituteMultiError(errors)
	}

	return nil
}

// RegexMatchAndSubstituteMultiError is an error wrapping multiple validation
// errors returned by RegexMatchAndSubstitute.ValidateAll() if the designated
// constraints aren't met.
type RegexMatchAndSubstituteMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m RegexMatchAndSubstituteMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m RegexMatchAndSubstituteMultiError) AllErrors() []error { return m }

// RegexMatchAndSubstituteValidationError is the validation error returned by
// RegexMatchAndSubstitute.Validate if the designated constraints aren't met.
type RegexMatchAndSubstituteValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e RegexMatchAndSubstituteValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e RegexMatchAndSubstituteValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e RegexMatchAndSubstituteValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e RegexMatchAndSubstituteValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e RegexMatchAndSubstituteValidationError) ErrorName() string {
	return "RegexMatchAndSubstituteValidationError"
}

// Error satisfies the builtin error interface
func (e RegexMatchAndSubstituteValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sRegexMatchAndSubstitute.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = RegexMatchAndSubstituteValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = RegexMatchAndSubstituteValidationError{}

var _RegexMatchAndSubstitute_Substitution_Pattern = regexp.MustCompile("^[^\x00\n\r]*$")

// Validate checks the field values on RegexMatcher_GoogleRE2 with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *RegexMatcher_GoogleRE2) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on RegexMatcher_GoogleRE2 with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// RegexMatcher_GoogleRE2MultiError, or nil if none found.
func (m *RegexMatcher_GoogleRE2) ValidateAll() error {
	return m.validate(true)
}

func (m *RegexMatcher_GoogleRE2) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if all {
		switch v := interface{}(m.GetMaxProgramSize()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, RegexMatcher_GoogleRE2ValidationError{
					field:  "MaxProgramSize",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, RegexMatcher_GoogleRE2ValidationError{
					field:  "MaxProgramSize",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetMaxProgramSize()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return RegexMatcher_GoogleRE2ValidationError{
				field:  "MaxProgramSize",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return RegexMatcher_GoogleRE2MultiError(errors)
	}

	return nil
}

// RegexMatcher_GoogleRE2MultiError is an error wrapping multiple validation
// errors returned by RegexMatcher_GoogleRE2.ValidateAll() if the designated
// constraints aren't met.
type RegexMatcher_GoogleRE2MultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m RegexMatcher_GoogleRE2MultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m RegexMatcher_GoogleRE2MultiError) AllErrors() []error { return m }

// RegexMatcher_GoogleRE2ValidationError is the validation error returned by
// RegexMatcher_GoogleRE2.Validate if the designated constraints aren't met.
type RegexMatcher_GoogleRE2ValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e RegexMatcher_GoogleRE2ValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e RegexMatcher_GoogleRE2ValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e RegexMatcher_GoogleRE2ValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e RegexMatcher_GoogleRE2ValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e RegexMatcher_GoogleRE2ValidationError) ErrorName() string {
	return "RegexMatcher_GoogleRE2ValidationError"
}

// Error satisfies the builtin error interface
func (e RegexMatcher_GoogleRE2ValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sRegexMatcher_GoogleRE2.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = RegexMatcher_GoogleRE2ValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = RegexMatcher_GoogleRE2ValidationError{}
