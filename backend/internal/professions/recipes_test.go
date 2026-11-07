package professions_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

type recipeJSON struct {
	ID         int             `json:"id"`
	Name       string          `json:"name"`
	Profession string          `json:"profession"`
	Characters []characterJSON `json:"characters"`
}

// Aeliana knows Light Armor Kit and Crispy Bat Wing; Zephyrion knows Light Armor Kit
// and Heavy Linen Bandage; Quitter, who left the guild, knows Linen Bandage.
const seedRecipes = `
	INSERT INTO characters (id, name, secondary_name, realm, class, role, left_guild_at)
	VALUES ('33333333-3333-3333-3333-333333333333', 'Quitter', 'Gone', 'Emberreach', 'Mage', 'dps', now());
	INSERT INTO recipes (id, profession, name)
	VALUES
		(1356, 'Leatherworking', 'Light Armor Kit'),
		(8616, 'Cooking', 'Crispy Bat Wing'),
		(1839, 'First Aid', 'Heavy Linen Bandage'),
		(1838, 'First Aid', 'Linen Bandage');
	INSERT INTO character_recipes (character_id, recipe_id)
	VALUES
		('11111111-1111-1111-1111-111111111111', 1356),
		('11111111-1111-1111-1111-111111111111', 8616),
		('22222222-2222-2222-2222-222222222222', 1356),
		('22222222-2222-2222-2222-222222222222', 1839),
		('33333333-3333-3333-3333-333333333333', 1838)`

var (
	lightArmorKit = recipeJSON{ID: 1356, Name: "Light Armor Kit", Profession: "Leatherworking", Characters: []characterJSON{aeliana, zephyrion}}
	crispyBatWing = recipeJSON{ID: 8616, Name: "Crispy Bat Wing", Profession: "Cooking", Characters: []characterJSON{aeliana}}
	heavyBandage  = recipeJSON{ID: 1839, Name: "Heavy Linen Bandage", Profession: "First Aid", Characters: []characterJSON{zephyrion}}
)

func TestListRecipes_ReturnsRecipesWithTheCharactersWhoKnowThem(t *testing.T) {
	// given recipes known by different characters, one known only by a character who left the guild
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(seedCharacters)
	db.MustExec(seedRecipes)

	// when I ask for the recipes
	resp := srv.Get(t, "/api/recipes")

	// then I expect each recipe with its crafters sorted by name, and nothing for the recipe only the leaver knows
	resp.RequireStatus(t, 200)
	var recipes []recipeJSON
	resp.DecodeJSON(t, &recipes)
	want := []recipeJSON{crispyBatWing, heavyBandage, lightArmorKit}
	if diff := cmp.Diff(want, recipes); diff != "" {
		t.Errorf("unexpected recipes (-want +got):\n%s", diff)
	}
}

func TestListRecipes_ReturnsMatchingRecipes_WhenSearchingByNameIgnoringCase(t *testing.T) {
	// given recipes whose names contain "bandage" and "armor"
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(seedCharacters)
	db.MustExec(seedRecipes)

	// when I search for "ARMOR"
	resp := srv.Get(t, "/api/recipes?q=ARMOR")

	// then I expect only Light Armor Kit
	resp.RequireStatus(t, 200)
	var recipes []recipeJSON
	resp.DecodeJSON(t, &recipes)
	if diff := cmp.Diff([]recipeJSON{lightArmorKit}, recipes); diff != "" {
		t.Errorf("unexpected recipes (-want +got):\n%s", diff)
	}
}

func TestListRecipes_ReturnsOnlyThatCharactersRecipes_WhenFilteringByCharacter(t *testing.T) {
	// given recipes known by two characters
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(seedCharacters)
	db.MustExec(seedRecipes)

	// when I filter by Zephyrion
	resp := srv.Get(t, "/api/recipes?characterId="+zephyrion.ID)

	// then I expect his two recipes, each still listing everyone who can craft it
	resp.RequireStatus(t, 200)
	var recipes []recipeJSON
	resp.DecodeJSON(t, &recipes)
	if diff := cmp.Diff([]recipeJSON{heavyBandage, lightArmorKit}, recipes); diff != "" {
		t.Errorf("unexpected recipes (-want +got):\n%s", diff)
	}
}

func TestListRecipes_ReturnsEmptyList_WhenNoRecipeMatches(t *testing.T) {
	// given recipes that do not match the search
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(seedCharacters)
	db.MustExec(seedRecipes)

	// when I search for something nobody can craft
	resp := srv.Get(t, "/api/recipes?q=dragonscale")

	// then I expect an empty JSON array, not null
	resp.RequireStatus(t, 200)
	if got := string(resp.Body); got != "[]\n" {
		t.Errorf("body = %q, want an empty array", got)
	}
}

func TestListRecipes_ReturnsBadRequest_WhenCharacterIdIsNotAUUID(t *testing.T) {
	// given any state
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I filter by a character id that is not a UUID
	resp := srv.Get(t, "/api/recipes?characterId=nobody")

	// then I expect a 400
	resp.RequireStatus(t, 400)
}
