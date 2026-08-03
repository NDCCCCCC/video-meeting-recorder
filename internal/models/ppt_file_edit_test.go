package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPPTFileEditFields(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(&PPTFile{})
	require.NoError(t, err)

	// Test 1: Create PPTFile with edit fields
	t.Run("Create with edit fields", func(t *testing.T) {
		ppt := PPTFile{
			FileName:      "test.pptx",
			FilePath:      "/path/to/test.pptx",
			BackupPath:    "/path/to/backup.pptx",
			DeletedSlides: `[1,5,10]`,
			EditHistory:   `[{"operation":"delete","slides":[1,5],"timestamp":"2026-04-20T00:00:00Z"}]`,
		}
		result := db.Create(&ppt)
		require.NoError(t, result.Error)

		// Verify fields saved
		var retrieved PPTFile
		db.First(&retrieved, ppt.ID)
		assert.Equal(t, "/path/to/backup.pptx", retrieved.BackupPath)
		assert.Equal(t, `[1,5,10]`, retrieved.DeletedSlides)
		assert.Equal(t, `[{"operation":"delete","slides":[1,5],"timestamp":"2026-04-20T00:00:00Z"}]`, retrieved.EditHistory)
	})

	// Test 2: HasBackup helper
	t.Run("HasBackup helper", func(t *testing.T) {
		ppt := PPTFile{
			FileName:   "test2.pptx",
			FilePath:   "/path/to/test2.pptx",
			BackupPath: "/path/to/backup.pptx",
		}
		db.Create(&ppt)

		assert.True(t, ppt.HasBackup())

		// Test when no backup
		ppt2 := PPTFile{
			FileName: "test3.pptx",
			FilePath: "/path/to/test3.pptx",
		}
		db.Create(&ppt2)
		assert.False(t, ppt2.HasBackup())
	})

	// Test 3: GetDeletedSlides helper
	t.Run("GetDeletedSlides helper", func(t *testing.T) {
		slidesJSON := `[1,5,10,15]`
		ppt := PPTFile{
			FileName:      "test4.pptx",
			FilePath:      "/path/to/test4.pptx",
			DeletedSlides: slidesJSON,
		}
		db.Create(&ppt)

		slides, err := ppt.GetDeletedSlides()
		require.NoError(t, err)
		assert.Equal(t, []int{1, 5, 10, 15}, slides)

		// Test empty JSON
		ppt2 := PPTFile{
			FileName:      "test5.pptx",
			FilePath:      "/path/to/test5.pptx",
			DeletedSlides: "[]",
		}
		db.Create(&ppt2)

		slides2, err := ppt2.GetDeletedSlides()
		require.NoError(t, err)
		assert.Equal(t, []int{}, slides2)

		// Test invalid JSON
		ppt3 := PPTFile{
			FileName:      "test6.pptx",
			FilePath:      "/path/to/test6.pptx",
			DeletedSlides: "invalid",
		}
		db.Create(&ppt3)

		slides3, err := ppt3.GetDeletedSlides()
		assert.Error(t, err)
		assert.Nil(t, slides3)
	})

	// Test 4: RecordDeletion helper
	t.Run("RecordDeletion helper", func(t *testing.T) {
		ppt := PPTFile{
			FileName:      "test7.pptx",
			FilePath:      "/path/to/test7.pptx",
			DeletedSlides: "[]",
		}
		db.Create(&ppt)

		// Record first deletion
		err := ppt.RecordDeletion([]int{1, 5})
		require.NoError(t, err)
		db.Save(&ppt)

		var retrieved PPTFile
		db.First(&retrieved, ppt.ID)
		assert.Equal(t, `[1,5]`, retrieved.DeletedSlides)

		// Record second deletion (should append)
		err = retrieved.RecordDeletion([]int{10, 15})
		require.NoError(t, err)
		db.Save(&retrieved)

		var retrieved2 PPTFile
		db.First(&retrieved2, ppt.ID)
		assert.Equal(t, `[1,5,10,15]`, retrieved2.DeletedSlides)
	})

	// Test 5: AddEditOperation helper
	t.Run("AddEditOperation helper", func(t *testing.T) {
		ppt := PPTFile{
			FileName:    "test8.pptx",
			FilePath:    "/path/to/test8.pptx",
			EditHistory: "[]",
		}
		db.Create(&ppt)

		// Add first operation
		err := ppt.AddEditOperation("delete", []int{1, 5})
		require.NoError(t, err)
		db.Save(&ppt)

		var retrieved PPTFile
		db.First(&retrieved, ppt.ID)

		// Parse and verify
		var history []EditOperation
		err = json.Unmarshal([]byte(retrieved.EditHistory), &history)
		require.NoError(t, err)
		assert.Len(t, history, 1)
		assert.Equal(t, "delete", history[0].Operation)
		assert.Equal(t, []int{1, 5}, history[0].Slides)
		assert.NotEmpty(t, history[0].Timestamp)

		// Add second operation
		err = retrieved.AddEditOperation("delete", []int{10, 15})
		require.NoError(t, err)
		db.Save(&retrieved)

		var retrieved2 PPTFile
		db.First(&retrieved2, ppt.ID)

		var history2 []EditOperation
		err = json.Unmarshal([]byte(retrieved2.EditHistory), &history2)
		require.NoError(t, err)
		assert.Len(t, history2, 2)
		assert.Equal(t, "delete", history2[1].Operation)
		assert.Equal(t, []int{10, 15}, history2[1].Slides)
	})
}
