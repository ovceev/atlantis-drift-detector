function downloadReport() {
    window.location.href = "/drift-detector/download-reports";
}

const ctx = document.getElementById('myChart').getContext('2d');

let myChart = new Chart(ctx, {
    type: 'bar',
    data: {
        labels: ['Errors', 'Drifted', 'No changes'],
        datasets: [{
            // Removing the label field from here
            data: [errorCount, driftedCount, noChangesCount],
            backgroundColor: [
                'rgba(255, 99, 132, 0.2)',
                'rgba(255, 204, 0, 0.2)',
                'rgba(75, 192, 192, 0.2)'
            ],
            borderColor: [
                'rgba(255, 99, 132, 1)',
                'rgba(255, 204, 0, 1)',
                'rgba(75, 192, 192, 1)'
            ],
            borderWidth: 1
        }]
    },
    options: {
        plugins: {
            legend: {
                display: false
            }
        },
        scales: {
            y: {
                beginAtZero: true
            }
        }
    }
});

function toggleChildren(event) {
    const folderDiv = event.currentTarget;
    const folderIcon = folderDiv.querySelector('.fas');
    const childrenDiv = folderDiv.nextElementSibling;

    // Toggle children display and folder icon
    if (childrenDiv.style.display === 'block') {
        childrenDiv.style.display = 'none';
        folderIcon.classList.add('fa-folder');
        folderIcon.classList.remove('fa-folder-open');
    } else {
        childrenDiv.style.display = 'block';
        folderIcon.classList.remove('fa-folder');
        folderIcon.classList.add('fa-folder-open');
    }

    // Preventing the event from reaching parent folders
    event.stopPropagation();

}
