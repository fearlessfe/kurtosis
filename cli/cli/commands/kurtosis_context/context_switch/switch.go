package context_switch

import (
	"context"
	"fmt"
	"github.com/kurtosis-tech/kurtosis-portal/api/golang/constructors"
	"github.com/kurtosis-tech/kurtosis/api/golang/engine/lib/kurtosis_context"
	"github.com/kurtosis-tech/kurtosis/cli/cli/command_framework/highlevel/context_id_arg"
	"github.com/kurtosis-tech/kurtosis/cli/cli/command_framework/lowlevel"
	"github.com/kurtosis-tech/kurtosis/cli/cli/command_framework/lowlevel/args"
	"github.com/kurtosis-tech/kurtosis/cli/cli/command_framework/lowlevel/flags"
	"github.com/kurtosis-tech/kurtosis/cli/cli/command_str_consts"
	"github.com/kurtosis-tech/kurtosis/contexts-config-store/store"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
)

const (
	contextIdentifierArgKey      = "context"
	contextIdentifierArgIsGreedy = false

	acceptNilResultForLocalClient = true
)

var ContextSwitchCmd = &lowlevel.LowlevelKurtosisCommand{
	CommandStr:       command_str_consts.ContextSwitchCmdStr,
	ShortDescription: "Switches to a different Kurtosis context",
	LongDescription: fmt.Sprintf("Switches to a different Kurtosis context. The context needs to be added "+
		"first using the `%s` command. When switching to a remote context, the connection will be established with "+
		"the remote Kurtosis server. Kurtosis Portal needs to be running for this. If the remote server can't be "+
		"reached, the context will remain unchanged.", command_str_consts.ContextAddCmdStr),
	Flags: []*flags.FlagConfig{},
	Args: []*args.ArgConfig{
		context_id_arg.NewContextIdentifierArg(store.GetContextsConfigStore(), contextIdentifierArgKey, contextIdentifierArgIsGreedy),
	},
	PreValidationAndRunFunc:  nil,
	RunFunc:                  run,
	PostValidationAndRunFunc: nil,
}

func run(ctx context.Context, _ *flags.ParsedFlags, args *args.ParsedArgs) error {
	contextIdentifier, err := args.GetNonGreedyArg(contextIdentifierArgKey)
	if err != nil {
		return stacktrace.Propagate(err, "Expected a value for context identifier arg '%v' but none was found; this is a bug with Kurtosis!", contextIdentifierArgKey)
	}

	isContextSwitchSuccessful := false
	logrus.Info("Switching context...")

	contextsConfigStore := store.GetContextsConfigStore()
	contextPriorToSwitch, err := contextsConfigStore.GetCurrentContext()
	if err != nil {
		return stacktrace.NewError("An error occurred retrieving current context prior to switching to the new one '%s'", contextIdentifier)
	}

	contextsMatchingIdentifiers, err := context_id_arg.GetContextUuidForContextIdentifier(contextsConfigStore, []string{contextIdentifier})
	if err != nil {
		return stacktrace.Propagate(err, "Error searching for context matching context identifier: '%s'", contextIdentifier)
	}
	contextUuidToSwitchTo, found := contextsMatchingIdentifiers[contextIdentifier]
	if !found {
		return stacktrace.NewError("No context matching identifier '%s' could be found", contextIdentifier)
	}

	if err = contextsConfigStore.SwitchContext(contextUuidToSwitchTo); err != nil {
		return stacktrace.Propagate(err, "An error occurred switching to context '%s' with UUID '%s'", contextIdentifier, contextUuidToSwitchTo.GetValue())
	}
	defer func() {
		if isContextSwitchSuccessful {
			return
		}
		if err = contextsConfigStore.SwitchContext(contextPriorToSwitch.GetUuid()); err != nil {
			logrus.Errorf("An unexpected error occurred switching to context '%s' with UUID "+
				"'%s'. Kurtosis tried to roll back to previous context '%s' with UUID '%s' but the roll back "+
				"failed. It is likely that the current context is still set to '%s' but it is not fully functional. "+
				"Try manually switching back to '%s' to get back to a working state. Error was:\n%v",
				contextIdentifier, contextUuidToSwitchTo.GetValue(), contextPriorToSwitch.GetName(),
				contextPriorToSwitch.GetUuid(), contextIdentifier, contextPriorToSwitch.GetName(), err.Error())
		}
	}()

	currentContext, err := contextsConfigStore.GetCurrentContext()
	if err != nil {
		return stacktrace.Propagate(err, "Error retrieving context info for context '%s' after switching to it", contextIdentifier)
	}
	portalDaemonClient, err := kurtosis_context.CreatePortalDaemonClient(currentContext, acceptNilResultForLocalClient)
	if err != nil {
		return stacktrace.Propagate(err, "Error connecting to Kurtosis portal after switching to the context'%s'", contextIdentifier)
	}

	if portalDaemonClient != nil {
		switchContextArg := constructors.NewSwitchContextArgs()
		if _, err = portalDaemonClient.SwitchContext(ctx, switchContextArg); err != nil {
			return stacktrace.Propagate(err, "Error switching Kurtosis portal context")
		}
	}

	logrus.Infof("Successfully switched to context '%s'", contextIdentifier)
	isContextSwitchSuccessful = true
	return nil
}