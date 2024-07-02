package loader

import (
	"context"
	"os"
	"testing"

	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/hexops/autogold/v2"
	"github.com/stretchr/testify/require"
)

func TestLoadOpenAPI(t *testing.T) {
	numOpenAPITools := func(set types.ToolSet) int {
		num := 0
		for _, v := range set {
			if v.IsOpenAPI() {
				num++
			}
		}
		return num
	}

	prgv3 := types.Program{
		ToolSet: types.ToolSet{},
	}
	datav3, err := os.ReadFile("testdata/openapi_v3.yaml")
	require.NoError(t, err)
	_, err = readTool(context.Background(), nil, &prgv3, &source{Content: datav3}, "")
	require.NoError(t, err, "failed to read openapi v3")
	require.Equal(t, 3, numOpenAPITools(prgv3.ToolSet), "expected 3 openapi tools")

	prgv2json := types.Program{
		ToolSet: types.ToolSet{},
	}
	datav2, err := os.ReadFile("testdata/openapi_v2.json")
	require.NoError(t, err)
	_, err = readTool(context.Background(), nil, &prgv2json, &source{Content: datav2}, "")
	require.NoError(t, err, "failed to read openapi v2")
	require.Equal(t, 3, numOpenAPITools(prgv2json.ToolSet), "expected 3 openapi tools")

	prgv2yaml := types.Program{
		ToolSet: types.ToolSet{},
	}
	datav2, err = os.ReadFile("testdata/openapi_v2.yaml")
	require.NoError(t, err)
	_, err = readTool(context.Background(), nil, &prgv2yaml, &source{Content: datav2}, "")
	require.NoError(t, err, "failed to read openapi v2 (yaml)")
	require.Equal(t, 3, numOpenAPITools(prgv2yaml.ToolSet), "expected 3 openapi tools")

	require.EqualValuesf(t, prgv2json.ToolSet, prgv2yaml.ToolSet, "expected same toolset for openapi v2 json and yaml")
}

func TestOpenAPIv3(t *testing.T) {
	prgv3 := types.Program{
		ToolSet: types.ToolSet{},
	}
	datav3, err := os.ReadFile("testdata/openapi_v3.yaml")
	require.NoError(t, err)
	_, err = readTool(context.Background(), nil, &prgv3, &source{Content: datav3}, "")

	autogold.ExpectFile(t, prgv3.ToolSet, autogold.Dir("testdata/openapi"))
}

func TestOpenAPIv3NoOperationIDs(t *testing.T) {
	prgv3 := types.Program{
		ToolSet: types.ToolSet{},
	}
	datav3, err := os.ReadFile("testdata/openapi_v3_no_operation_ids.yaml")
	require.NoError(t, err)
	_, err = readTool(context.Background(), nil, &prgv3, &source{Content: datav3}, "")

	autogold.ExpectFile(t, prgv3.ToolSet, autogold.Dir("testdata/openapi"))
}

func TestOpenAPIv2(t *testing.T) {
	prgv2 := types.Program{
		ToolSet: types.ToolSet{},
	}
	datav2, err := os.ReadFile("testdata/openapi_v2.yaml")
	require.NoError(t, err)
	_, err = readTool(context.Background(), nil, &prgv2, &source{Content: datav2}, "")

	autogold.ExpectFile(t, prgv2.ToolSet, autogold.Dir("testdata/openapi"))
}
