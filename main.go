package main

import (
	"encoding/json"
	"fmt"
	"log"      // <-- MAKE SURE THIS IS HERE
	"net/http" // <-- MAKE SURE THIS IS HERE
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
	filename  = "tasks.json" // <-- MAKE SURE THIS IS INSIDE THE VAR BLOCK
)

func addTask(title, desc, priority string) {
	mutex.Lock()
	defer mutex.Unlock()

	uniqueID := fmt.Sprintf("TASK-%d", os.Getpid()+len(boardData.Tasks))

	newTask := Task{
		ID:          uniqueID,
		Title:       title,
		Description: desc,
		Status:      "Backlog", // All new tasks start in the Backlog!
		Priority:    priority,
	}

	boardData.Tasks = append(boardData.Tasks, newTask)

	saveFile()
}

func deleteTask(idToKill string) {
	mutex.Lock()
	defer mutex.Unlock()

	// 1. Create a blank slice to hold everything we want to KEEP
	var updatedTasks []Task

	// 2. Loop through every task currently on our board
	for _, task := range boardData.Tasks {
		// 3. If the task's ID DOES NOT match the ID we want to delete, KEEP IT!
		if task.ID != idToKill {
			updatedTasks = append(updatedTasks, task)
		}
		// If it DOES match, we simply skip it (which drops it from existence)
	}

	// 4. Overwrite our global board data with our filtered list
	boardData.Tasks = updatedTasks
	saveFile()
}

func saveFile() {
	bytes, _ := json.Marshal(boardData)
	os.WriteFile(filename, bytes, 0644)
}

func loadFile() {
	bytes, err := os.ReadFile(filename)
	if err == nil {
		json.Unmarshal(bytes, &boardData)
	}
	// Safety net: If the file was empty, make sure the slice isn't nil
	if boardData.Tasks == nil {
		boardData.Tasks = []Task{}
	}
}

func moveTask(idToMove string, newStatus string) {
	mutex.Lock()
	defer mutex.Unlock()

	// Loop through our list to find the exact task by its unique fingerprint (ID)
	for i := 0; i < len(boardData.Tasks); i++ {
		if boardData.Tasks[i].ID == idToMove {
			// Found it! Overwrite just its Status field
			boardData.Tasks[i].Status = newStatus
			break // Stop looping immediately, our mission is complete
		}
	}
	saveFile()
}

func main() {
	// Auto-create storage file if missing
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		initialData := []byte(`{"tasks":[]}`)
		os.WriteFile(filename, initialData, 0644)
	}

	loadFile()

	// --- ROUTE 1: CREATING A TASK ---
	// URL Example: /add-task?title=Fix+Bugs&desc=Clean+the+code&priority=High
	http.HandleFunc("/add-task", func(w http.ResponseWriter, r *http.Request) {
		title := r.URL.Query().Get("title")
		desc := r.URL.Query().Get("desc")
		priority := r.URL.Query().Get("priority")

		if title != "" {
			addTask(title, desc, priority)
		}
		// Redirect back to the main dashboard interface
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// --- ROUTE 2: DELETING A TASK ---
	// URL Example: /delete-task?id=TASK-101
	http.HandleFunc("/delete-task", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id != "" {
			deleteTask(id)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// --- ROUTE 3: MOVING A TASK ---
	// URL Example: /move-task?id=TASK-101&status=In+Progress
	http.HandleFunc("/move-task", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		status := r.URL.Query().Get("status")

		if id != "" && status != "" {
			moveTask(id, status)
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
		// We define the statuses we want to display on our board
		statuses := []string{"Backlog", "In Progress", "Done"}

		for _, currentStatus := range statuses {
			// Start a column container
			fmt.Fprintf(w, "<div class='column'><h2>%s</h2>", currentStatus)

			// Loop through all tasks and display only the ones matching this column's status
			for _, task := range boardData.Tasks {
				if task.Status == currentStatus {
					fmt.Fprintf(w, "<div class='task-card priority-%s'>", task.Priority)
					fmt.Fprintf(w, "<div class='task-title'>%s <span style='font-size:10px; color:#666;'>[%s]</span></div>", task.Title, task.Priority)
					fmt.Fprintf(w, "<div class='task-desc'>%s</div>", task.Description)

					// Draw context-aware movement buttons depending on where the task currently lives!
					if task.Status == "Backlog" {
						fmt.Fprintf(w, "<button class='btn btn-move' onclick=\"window.location.href='/move-task?id=%s&status=In+Progress'\">🚀 Start</button>", task.ID)
					}
					if task.Status == "In Progress" {
						fmt.Fprintf(w, "<button class='btn btn-move' onclick=\"window.location.href='/move-task?id=%s&status=Done'\">✅ Finish</button>", task.ID)
					}

					// Every single card gets a delete option
					fmt.Fprintf(w, "<button class='btn btn-delete' onclick=\"if(confirm('Delete task?')) window.location.href='/delete-task?id=%s'\">🗑️ Delete</button>", task.ID)

					fmt.Fprint(w, "</div>") // Close task-card
				}
			}

			fmt.Fprint(w, "</div>") // Close column
		}

		fmt.Fprint(w, "</div>") // Close board

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
