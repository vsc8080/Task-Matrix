package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
)

// This is the blueprint for a single task item
type Task struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
}

// This is the blueprint for our entire storage file
type BoardData struct {
	Tasks []Task `json:"tasks"`
}

// These are our global variables
var (
	boardData BoardData
	mutex     sync.Mutex
	filename  = "tasks.json"
)

func addTask(title, desc, priority string) {
	mutex.Lock()
	defer mutex.Unlock()

	uniqueID := fmt.Sprintf("TASK-%d", os.Getpid()+len(boardData.Tasks))

	newTask := Task{
		ID:          uniqueID,
		Title:       title,
		Description: desc,
		Status:      "Backlog",
		Priority:    priority,
	}

	boardData.Tasks = append(boardData.Tasks, newTask)
	saveFile()
}

func deleteTask(idToKill string) {
	mutex.Lock()
	defer mutex.Unlock()

	var updatedTasks []Task
	for _, task := range boardData.Tasks {
		if task.ID != idToKill {
			updatedTasks = append(updatedTasks, task)
		}
	}

	boardData.Tasks = updatedTasks
	saveFile()
}

// saveFile returns an error instead of hiding it
func saveFile() error {
	bytes, err := json.Marshal(boardData)
	if err != nil {
		return fmt.Errorf("failed to marshal board data: %w", err)
	}

	err = os.WriteFile(filename, bytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write matrix file: %w", err)
	}

	return nil
}

// loadFile alerts us if the data on disk is corrupted
func loadFile() error {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			boardData.Tasks = []Task{}
			return nil
		}
		return fmt.Errorf("failed to read storage file: %w", err)
	}

	err = json.Unmarshal(bytes, &boardData)
	if err != nil {
		return fmt.Errorf("file is corrupted or invalid JSON: %w", err)
	}

	if boardData.Tasks == nil {
		boardData.Tasks = []Task{}
	}

	return nil
}

// Upgraded moveTask function (declared cleanly at the root level!)
func moveTask(idToMove string, newStatus string) bool {
	mutex.Lock()
	defer mutex.Unlock()

	found := false
	for i := 0; i < len(boardData.Tasks); i++ {
		if boardData.Tasks[i].ID == idToMove {
			boardData.Tasks[i].Status = newStatus
			found = true
			break
		}
	}

	if found {
		saveFile()
	}
	return found
}

func main() {
	// Auto-create storage file if missing
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		initialData := []byte(`{"tasks":[]}`)
		os.WriteFile(filename, initialData, 0644)
	}

	loadFile()

	// --- ROUTE 1: CREATING A TASK (WITH ERROR MANAGEMENT) ---
	http.HandleFunc("/add-task", func(w http.ResponseWriter, r *http.Request) {
		title := r.URL.Query().Get("title")
		desc := r.URL.Query().Get("desc")
		priority := r.URL.Query().Get("priority")

		if title == "" {
			http.Error(w, "Missing required parameter: title", http.StatusBadRequest)
			return
		}

		addTask(title, desc, priority)

		if err := saveFile(); err != nil {
			log.Printf("[ERROR] AddTask state save failed: %v", err)
			http.Error(w, "Database write failure", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// --- ROUTE 2: DELETING A TASK ---
	http.HandleFunc("/delete-task", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id != "" {
			deleteTask(id)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// --- ROUTE 3: MOVING A TASK (WITH NOT FOUND VALIDATION) ---
	http.HandleFunc("/move-task", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		status := r.URL.Query().Get("status")

		if id == "" || status == "" {
			http.Error(w, "Missing target id or status", http.StatusBadRequest)
			return
		}

		wasMoved := moveTask(id, status)
		if !wasMoved {
			http.Error(w, fmt.Sprintf("Task with ID %s not found on matrix", id), http.StatusNotFound)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// --- ROUTE 4: THE MATRIX DASHBOARD VIEW ---
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		// 1. Core Styles for our 3-Column Grid
		fmt.Fprint(w, `
            <html>
            <head>
                <title>Task Matrix Dashboard</title>
                <style>
                    body { font-family: sans-serif; background: #121212; color: white; text-align: center; margin: 0; padding: 20px; }
                    .board { display: flex; justify-content: center; gap: 20px; max-width: 1200px; margin: auto; align-items: flex-start; }
                    .column { background: #1e1e1e; border: 1px solid #333; border-radius: 8px; width: 30%; min-width: 250px; padding: 15px; box-sizing: border-box; }
                    .column h2 { margin-top: 0; padding-bottom: 10px; border-bottom: 2px solid #444; }
                    .task-card { background: #2d2d2d; border-left: 5px solid #666; padding: 12px; margin-bottom: 15px; border-radius: 4px; text-align: left; position: relative; }
                    .task-title { font-weight: bold; font-size: 16px; margin-bottom: 5px; }
                    .task-desc { color: #aaa; font-size: 13px; margin-bottom: 10px; }
                    
                    /* Priority Highlights */
                    .priority-High { border-left-color: #ff3333; }
                    .priority-Medium { border-left-color: #ffaa00; }
                    .priority-Low { border-left-color: #33cc33; }

                    .btn { cursor: pointer; padding: 4px 8px; font-size: 11px; border: none; border-radius: 3px; color: white; margin-right: 5px; }
                    .btn-move { background: #0088ff; }
                    .btn-delete { background: #cc3333; }
                    .form-container { background: #1e1e1e; max-width: 500px; margin: 30px auto; padding: 20px; border-radius: 8px; border: 1px solid #333; text-align: left; }
                    .form-group { margin-bottom: 12px; }
                    .form-group label { display: block; margin-bottom: 5px; font-size: 14px; }
                    .form-group input, .form-group textarea, .form-group select { width: 100%; padding: 8px; background: #2d2d2d; color: white; border: 1px solid #444; border-radius: 4px; box-sizing: border-box; }
                </style>
            </head>
            <body>
                <h1>Task Matrix Dashboard</h1>
                
                <div class="board">
        `)

		// 2. RENDERING THE 3 COLUMNS
		statuses := []string{"Backlog", "In Progress", "Done"}

		for _, currentStatus := range statuses {
			fmt.Fprintf(w, "<div class='column'><h2>%s</h2>", currentStatus)

			for _, task := range boardData.Tasks {
				if task.Status == currentStatus {
					fmt.Fprintf(w, "<div class='task-card priority-%s'>", task.Priority)
					fmt.Fprintf(w, "<div class='task-title'>%s <span style='font-size:10px; color:#666;'>[%s]</span></div>", task.Title, task.Priority)
					fmt.Fprintf(w, "<div class='task-desc'>%s</div>", task.Description)

					if task.Status == "Backlog" {
						fmt.Fprintf(w, "<button class='btn btn-move' onclick=\"window.location.href='/move-task?id=%s&status=In+Progress'\">🚀 Start</button>", task.ID)
					}
					if task.Status == "In Progress" {
						fmt.Fprintf(w, "<button class='btn btn-move' onclick=\"window.location.href='/move-task?id=%s&status=Done'\">✅ Finish</button>", task.ID)
					}

					fmt.Fprintf(w, "<button class='btn btn-delete' onclick=\"if(confirm('Delete task?')) window.location.href='/delete-task?id=%s'\">🗑️ Delete</button>", task.ID)

					fmt.Fprint(w, "</div>")
				}
			}

			fmt.Fprint(w, "</div>")
		}

		fmt.Fprint(w, "</div>")

		// 3. THE CREATION FORM LAYOUT
		fmt.Fprint(w, `
            <div class="form-container">
                <h3>Create New Matrix Task</h3>
                <form action="/add-task" method="GET">
                    <div class="form-group">
                        <label>Task Title</label>
                        <input type="text" name="title" required placeholder="What needs doing?">
                    </div>
                    <div class="form-group">
                        <label>Description</label>
                        <textarea name="desc" rows="2" placeholder="Add some details..."></textarea>
                    </div>
                    <div class="form-group">
                        <label>Priority Matrix</label>
                        <select name="priority">
                            <option value="Low">Low (Green)</option>
                            <option value="Medium" selected>Medium (Orange)</option>
                            <option value="High">High (Red)</option>
                        </select>
                    </div>
                    <button type="submit" style="background:#33cc33; color:white; padding:10px 15px; border:none; border-radius:4px; cursor:pointer; font-weight:bold; width:100%;">+ Insert Into Matrix</button>
                </form>
            </div>
            </body>
            </html>
        `)
	})

	fmt.Println("Task Matrix server starting on http://localhost:8080 ...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
