package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/clock"

	"go.ytsaurus.tech/yt/go/ypath"
)

func getYtsaurus(t *testing.T, ytLocal *YtsaurusLocal) *Ytsaurus {
	require.NoError(t, ytLocal.Start())

	require.NoError(t, os.Setenv("YT_TOKEN", ytDevToken))
	yt, err := NewYtsaurus(
		&YtsaurusConfig{
			Proxy:               ytLocal.GetProxy(),
			Timeout:             10 * time.Minute,
			LogLevel:            "DEBUG",
			ApplyUserChanges:    true,
			ApplyGroupChanges:   true,
			ApplyMemberChanges:  true,
			SourceAttributeName: "azure",
		}, getDevelopmentLogger(),
		clock.RealClock{},
	)
	require.NoError(t, err)
	return yt
}

// TestUpdateUserFirstName is a case for the  specific bug.
// If same `name` value is passed in multiset_attributes request for user update
// YTsaurus will update other attributes, but will fail with 501 error.
// Since fields are updated this bug doesn't have consequences, though it is nice not to have
// scary errors in logs.
func TestUpdateUserFirstName(t *testing.T) {
	ytLocal := NewYtsaurusLocal()
	defer func() { require.NoError(t, ytLocal.Stop()) }()
	yt := getYtsaurus(t, ytLocal)

	const azureID = "fake-az-id-old"

	managedOleg := YtsaurusUser{
		Username: "oleg",
		SourceRaw: map[string]any{
			"id":         azureID,
			"first_name": "Lego",
		},
	}
	err := yt.CreateUser(managedOleg)
	require.NoError(t, err)

	managedOleg.SourceRaw = map[string]any{
		"id":         azureID,
		"first_name": "Oleg",
	}

	updErr := yt.UpdateUser(managedOleg.Username, managedOleg)

	ytClient, err := ytLocal.GetClient()
	require.NoError(t, err)

	var updatedName string
	err = ytClient.GetNode(
		context.Background(),
		ypath.Path("//sys/users/"+managedOleg.Username+"/@azure/first_name"),
		&updatedName,
		nil,
	)
	require.NoError(t, err)

	require.Equal(t, updatedName, "Oleg")
	require.NoError(t, updErr)
}

func TestGroups(t *testing.T) {
	ytLocal := NewYtsaurusLocal()
	defer func() { require.NoError(t, ytLocal.Stop()) }()
	yt := getYtsaurus(t, ytLocal)

	groupsInitial, err := yt.GetGroupsWithMembers()
	require.NoError(t, err)
	require.Empty(t, groupsInitial)

	managedOleg := YtsaurusUser{
		Username: "oleg",
		SourceRaw: map[string]any{
			"id": "fake-az-id-oleg",
		},
	}
	err = yt.CreateUser(managedOleg)
	require.NoError(t, err)

	managedOlegsGroup := YtsaurusGroup{
		Name: "olegs",
		SourceRaw: map[string]any{
			"id":           "fake-az-id-olegs",
			"display_name": "This is group is for Olegs only",
		},
	}
	err = yt.CreateGroup(managedOlegsGroup)
	require.NoError(t, err)

	err = yt.AddMember(managedOleg.Username, managedOlegsGroup.Name)
	require.NoError(t, err)

	groupsAfterCreate, err := yt.GetGroupsWithMembers()
	require.NoError(t, err)
	members := NewStringSet()
	members.Add(managedOleg.Username)
	require.Equal(t, []YtsaurusGroupWithMembers{
		{
			YtsaurusGroup: YtsaurusGroup{
				Name: managedOlegsGroup.Name,
				SourceRaw: map[string]any{
					"id":           managedOlegsGroup.SourceRaw["id"],
					"display_name": managedOlegsGroup.SourceRaw["display_name"],
				},
			},
			Members: members,
		},
	}, groupsAfterCreate)

	err = yt.RemoveMember(managedOleg.Username, managedOlegsGroup.Name)
	require.NoError(t, err)

	groupsAfterRemoveMember, err := yt.GetGroupsWithMembers()
	require.NoError(t, err)
	require.Equal(t, []YtsaurusGroupWithMembers{
		{
			YtsaurusGroup: YtsaurusGroup{
				Name: managedOlegsGroup.Name,
				SourceRaw: map[string]any{
					"id":           managedOlegsGroup.SourceRaw["id"],
					"display_name": managedOlegsGroup.SourceRaw["display_name"],
				},
			},
			Members: NewStringSet(),
		},
	}, groupsAfterRemoveMember)

	err = yt.RemoveGroup(managedOlegsGroup.Name)
	require.NoError(t, err)

	groupsAfterRemove, err := yt.GetGroupsWithMembers()
	require.NoError(t, err)
	require.Empty(t, groupsAfterRemove)

}
