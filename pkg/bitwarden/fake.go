package bitwarden

import (
	"fmt"
	"time"
)

type fakeClient struct {
	items       []Item
	attachments map[string]string
}

func (c fakeClient) GetFieldOnItem(itemName, fieldName string) ([]byte, error) {
	for _, item := range c.items {
		if itemName == item.Name {
			for _, field := range item.Fields {
				if field.Name == fieldName {
					return []byte(field.Value), nil
				}
			}
		}
	}
	return nil, fmt.Errorf("failed to find field %s in item %s", fieldName, itemName)
}

func (c fakeClient) GetAttachmentOnItem(itemName, attachmentName string) ([]byte, error) {
	for _, item := range c.items {
		if itemName == item.Name {
			for _, attachment := range item.Attachments {
				if attachment.FileName == attachmentName {
					if value, ok := c.attachments[attachment.ID]; ok {
						return []byte(value), nil
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("failed to find attachment %s in item %s", attachmentName, itemName)
}

func (c fakeClient) Logout() ([]byte, error) {
	return []byte("logged out"), nil
}

func (c fakeClient) GetPassword(itemName string) ([]byte, error) {
	for _, item := range c.items {
		if itemName == item.Name {
			if item.Login != nil {
				return []byte(item.Login.Password), nil
			}
		}
	}
	return nil, fmt.Errorf("failed to find password in item %s", itemName)
}

func getNewUUID() string {
	nanoTime := time.Now().Nanosecond()
	return fmt.Sprintf("%d", nanoTime)
}

func (c fakeClient) SetFieldOnItem(itemName, fieldName string, fieldValue []byte) error {
	var targetItem *Item
	var targetField *Field
	for index, item := range c.items {
		if itemName != item.Name {
			continue
		}
		targetItem = &c.items[index]
		for fieldIndex, field := range item.Fields {
			if field.Name == fieldName {
				targetField = &c.items[index].Fields[fieldIndex]
				break
			}
		}
		break

	}
	if targetItem == nil {
		newItemID := getNewUUID()
		c.items = append(c.items, Item{ID: newItemID, Name: itemName, Type: 1})
		targetItem = &c.items[len(c.items)-1]
	}
	if targetField == nil {
		targetItem.Fields = append(targetItem.Fields, Field{fieldName, string(fieldValue)})
		targetField = &targetItem.Fields[len(targetItem.Fields)-1]
	}
	targetField.Value = string(fieldValue)
	return nil
}

func (c fakeClient) SetAttachmentOnItem(itemName, attachmentName string, fileContents []byte) error {
	var targetItem *Item
	var targetAttachment *Attachment
	for index, item := range c.items {
		if itemName != item.Name {
			continue
		}
		targetItem = &c.items[index]
		for attachmentIndex, attachment := range item.Attachments {
			if attachment.FileName == attachmentName {
				targetAttachment = &c.items[index].Attachments[attachmentIndex]
				break
			}
		}
		break
	}
	if targetItem == nil {
		newItemID := getNewUUID()
		c.items = append(c.items, Item{ID: newItemID, Name: itemName, Type: 1})
		targetItem = &c.items[len(c.items)-1]
	}
	if targetAttachment == nil {
		newAttachmentID := getNewUUID()
		c.attachments[newAttachmentID] = string(fileContents)
		targetAttachment = &Attachment{newAttachmentID, attachmentName}
		targetItem.Attachments = append(targetItem.Attachments, *targetAttachment)
	}
	c.attachments[targetAttachment.ID] = string(fileContents)
	return nil
}

func (c fakeClient) SetPassword(itemName string, password []byte) error {
	var targetItem *Item
	for index, item := range c.items {
		if itemName == item.Name {
			targetItem = &c.items[index]
			break
		}
	}
	if targetItem == nil {
		newItemID := getNewUUID()
		c.items = append(c.items, Item{ID: newItemID, Name: itemName, Type: 1, Login: &Login{Password: string(password)}})
		targetItem = &c.items[len(c.items)-1]
	}
	targetItem.Login.Password = string(password)
	return nil
}

// NewFakeClient generates a fake BitWarden client which is supposed to used only for testing
func NewFakeClient(items []Item, attachments map[string]string) Client {
	return fakeClient{items: items, attachments: attachments}
}
