package test_engine

import (
	"fmt"
	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/service_network"
	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/startosis_engine/kurtosis_instruction/exec"
	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/startosis_engine/kurtosis_starlark_framework/kurtosis_plan_instruction"
	"github.com/kurtosis-tech/kurtosis/core/server/api_container/server/startosis_engine/runtime_value_store"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
	"testing"
)

type execTestCase struct {
	*testing.T
}

func newExecTestCase(t *testing.T) *execTestCase {
	return &execTestCase{
		T: t,
	}
}

func (t execTestCase) GetId() string {
	return exec.ExecBuiltinName
}

func (t execTestCase) GetInstruction() *kurtosis_plan_instruction.KurtosisPlanInstruction {
	serviceNetwork := service_network.NewMockServiceNetwork(t)
	runtimeValueStore := runtime_value_store.NewRuntimeValueStore()

	serviceNetwork.EXPECT().ExecCommand(
		mock.Anything,
		string(TestServiceName),
		[]string{"mkdir", "-p", "/tmp/store"},
	).Times(1).Return(
		int32(0),
		"",
		nil,
	)

	return exec.NewExec(serviceNetwork, runtimeValueStore)
}

func (t execTestCase) GetStarlarkCode() string {
	recipe := fmt.Sprintf(`ExecRecipe(service_name=%q, command=["mkdir", "-p", "/tmp/store"])`, TestServiceName)
	return fmt.Sprintf("%s(%s=%s)", exec.ExecBuiltinName, exec.RecipeArgName, recipe)
}

func (t execTestCase) Assert(interpretationResult starlark.Value, executionResult *string) {
	expectedInterpretationResultMap := `{"code": "{{kurtosis:[0-9a-f]{32}:code.runtime_value}}", "output": "{{kurtosis:[0-9a-f]{32}:output.runtime_value}}"}`
	require.Regexp(t, expectedInterpretationResultMap, interpretationResult.String())

	require.Equal(t, "Command returned with exit code '0' with no output", *executionResult)
}